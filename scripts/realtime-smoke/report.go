package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"go-boilerplate/pkg/xerrors"
)

const (
	// VerdictCompatible は、呼び出しが受理され事後条件も成立したことを示します。
	VerdictCompatible Verdict = "互換"
	// VerdictIncompatible は、呼び出しは受理されたが事後条件が成立しない、または production では
	// 通る呼び出しが拒否されたことを示します（silent drop / envelope 混入 / 順序違い を含む）。
	VerdictIncompatible Verdict = "非互換"
	// VerdictUnsupported は、エミュレータが未実装を明示して返したことを示します。
	VerdictUnsupported Verdict = "未対応"
	// VerdictUnverifiable は、transport 失敗・timeout・先行検査の失敗により判定に至らなかったことを示します。
	VerdictUnverifiable Verdict = "検証不能"

	// maxDetailLen は、表に載せるエラー本文の上限です。SDK のエラーは複数行になるため要約します。
	maxDetailLen = 200
)

var (
	// errNoResults は、検査が 1 件も記録されなかったときのエラーです。
	errNoResults = xerrors.New("no check was recorded")

	// unsupportedCodes は、エミュレータが未実装を明示するときに返すエラーコードです。
	unsupportedCodes = map[string]struct{}{
		"InvalidAction":        {},
		"NotImplemented":       {},
		"UnknownOperation":     {},
		"UnsupportedOperation": {},
		"MissingAction":        {},
	}

	// unsupportedMarkers は、未実装を示すメッセージ断片（小文字比較）です。SDK が JSON 応答を期待して
	// XML / HTML を受けたときの "deserialization failed"、経路自体が無いときの 404 / 501 を含みます。
	unsupportedMarkers = []string{
		"not implemented",
		"unsupported",
		"unknown operation",
		"deserialization failed",
		"statuscode: 404",
		"statuscode: 501",
		"statuscode: 405",
	}
)

// Verdict は、検査 1 件の判定です。
type Verdict string

// Result は、検査 1 件の記録です。
type Result struct {
	ID      string
	Subject string
	Check   string
	Verdict Verdict
	Detail  string
}

// apiError は、AWS SDK が返すサービスエラーのうちこのツールが読む部分です。
// smithy-go を direct 依存へ昇格させないため、インターフェースをここで再宣言します。
type apiError interface {
	error
	ErrorCode() string
	ErrorMessage() string
}

// incompatibleError は、呼び出し自体は成功したが事後条件が成立しなかったことを表します。
type incompatibleError struct {
	detail string
}

// step は、検査 1 件です。fn は成功時に表へ載せる詳細を返し、失敗はエラーで返します
// （事後条件の不成立は incompatible で包む）。halt は、失敗したら後続を実行不能にする検査を示します。
type step struct {
	id    string
	check string
	halt  bool
	fn    func(ctx context.Context) (string, error)
}

// recorder は、検査結果を順に蓄えます。
type recorder struct {
	results []Result
}

func (e *incompatibleError) Error() string { return e.detail }

// incompatible は、事後条件の不成立を Verdict 非互換として記録するためのエラーを返します。
func incompatible(detail string) error {
	return &incompatibleError{detail: detail}
}

// classify は、エラーから Verdict を導きます。transport の失敗だけを検証不能とし、
// サーバーが応答した失敗は未対応か非互換のどちらかに必ず倒します。
func classify(err error) Verdict {
	if err == nil {
		return VerdictCompatible
	}

	var inc *incompatibleError
	if xerrors.As(err, &inc) {
		return VerdictIncompatible
	}

	if isTransportFailure(err) {
		return VerdictUnverifiable
	}

	var api apiError
	if xerrors.As(err, &api) {
		if _, ok := unsupportedCodes[api.ErrorCode()]; ok {
			return VerdictUnsupported
		}
	}

	if hasUnsupportedMarker(err.Error()) {
		return VerdictUnsupported
	}

	return VerdictIncompatible
}

// isTransportFailure は、応答が返る前に失敗したかを判定します。
func isTransportFailure(err error) bool {
	if xerrors.Is(err, context.DeadlineExceeded) || xerrors.Is(err, context.Canceled) {
		return true
	}

	var netErr net.Error

	return xerrors.As(err, &netErr)
}

func hasUnsupportedMarker(msg string) bool {
	lower := strings.ToLower(msg)
	for _, m := range unsupportedMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}

	return false
}

// describe は、エラーを表の 1 セルに収まる形にします。
func describe(err error) string {
	var api apiError
	msg := err.Error()
	if xerrors.As(err, &api) {
		msg = api.ErrorCode() + ": " + api.ErrorMessage()
	}

	msg = strings.Join(strings.Fields(msg), " ")
	if len(msg) > maxDetailLen {
		msg = msg[:maxDetailLen] + "…"
	}

	return msg
}

func (r *recorder) add(id, subject, check string, verdict Verdict, detail string) {
	r.results = append(r.results, Result{ID: id, Subject: subject, Check: check, Verdict: verdict, Detail: detail})
}

// record は、検査の戻りを分類して記録し、判定を返します。
func (r *recorder) record(id, subject, check, detail string, err error) Verdict {
	verdict := classify(err)
	if err != nil {
		detail = describe(err)
	}

	r.add(id, subject, check, verdict, detail)

	return verdict
}

// skip は、先行検査の失敗などで実行に至らなかった検査を検証不能として残します。
// 黙って省くと「検査していない」が「クリーン」に化けるため、行として必ず出します。
func (r *recorder) skip(id, subject, check, reason string) {
	r.add(id, subject, check, VerdictUnverifiable, reason)
}

// runChain は、検査を順に実行して記録します。halt な検査が互換以外で終わったら残りを検証不能として
// 残し、先頭の検査が互換で終わったかを返します（後片付けの要否に使う）。
func runChain(ctx context.Context, subject string, steps []step, rec *recorder) bool {
	firstPassed := false
	for i, st := range steps {
		detail, err := st.fn(ctx)
		verdict := rec.record(st.id, subject, st.check, detail, err)
		if i == 0 {
			firstPassed = verdict == VerdictCompatible
		}

		if verdict != VerdictCompatible && st.halt {
			for _, rest := range steps[i+1:] {
				rec.skip(rest.id, subject, rest.check, "先行検査 "+st.id+" により実行不能")
			}

			break
		}
	}

	return firstPassed
}

// count は、Verdict ごとの件数を返します。
func count(results []Result) map[Verdict]int {
	c := map[Verdict]int{}
	for _, r := range results {
		c[r.Verdict]++
	}

	return c
}

// exitCode は、結果表からプロセスの終了コードを決めます。検証不能が 1 件でもあれば失敗、
// strict では非互換 / 未対応も失敗にします（CI に組み込むときの口）。
func exitCode(results []Result, strict bool) (int, error) {
	if len(results) == 0 {
		return 1, errNoResults
	}

	c := count(results)
	if c[VerdictUnverifiable] > 0 {
		return 1, nil
	}

	if strict && c[VerdictIncompatible]+c[VerdictUnsupported] > 0 {
		return 1, nil
	}

	return 0, nil
}

// writeMarkdown は、親 issue へ貼り付ける Markdown を書き出します。
func writeMarkdown(w io.Writer, results []Result) error {
	if _, err := fmt.Fprintln(w, "| # | Subject | Check | Verdict | Detail |"); err != nil {
		return xerrors.Wrap(err, "write header")
	}

	if _, err := fmt.Fprintln(w, "| --- | --- | --- | --- | --- |"); err != nil {
		return xerrors.Wrap(err, "write header")
	}

	for _, r := range results {
		if _, err := fmt.Fprintf(w, "| %s | %s | %s | **%s** | %s |\n",
			r.ID, r.Subject, r.Check, r.Verdict, escapeCell(r.Detail)); err != nil {
			return xerrors.Wrap(err, "write row")
		}
	}

	return writeSummary(w, results)
}

// writeSummary は、件数の要約と compatibility implementation 候補（非互換 / 未対応のみ）を書き出します。
func writeSummary(w io.Writer, results []Result) error {
	c := count(results)
	if _, err := fmt.Fprintf(w, "\n%s %d / %s %d / %s %d / %s %d\n",
		VerdictCompatible, c[VerdictCompatible], VerdictIncompatible, c[VerdictIncompatible],
		VerdictUnsupported, c[VerdictUnsupported], VerdictUnverifiable, c[VerdictUnverifiable]); err != nil {
		return xerrors.Wrap(err, "write summary")
	}

	if _, err := fmt.Fprintln(w, "\n### compatibility implementation 候補（非互換 / 未対応）"); err != nil {
		return xerrors.Wrap(err, "write summary")
	}

	found := false
	for _, r := range results {
		if r.Verdict != VerdictIncompatible && r.Verdict != VerdictUnsupported {
			continue
		}

		found = true
		if _, err := fmt.Fprintf(w, "- %s %s: %s — %s\n", r.ID, r.Check, r.Verdict, escapeCell(r.Detail)); err != nil {
			return xerrors.Wrap(err, "write candidate")
		}
	}

	if !found {
		if _, err := fmt.Fprintln(w, "- なし"); err != nil {
			return xerrors.Wrap(err, "write candidate")
		}
	}

	return nil
}

// writeText は、端末向けの整形なし出力を書き出します。
func writeText(w io.Writer, results []Result) error {
	for _, r := range results {
		if _, err := fmt.Fprintf(w, "%-4s %-14s %-42s %-6s %s\n", r.ID, r.Subject, r.Check, r.Verdict, r.Detail); err != nil {
			return xerrors.Wrap(err, "write row")
		}
	}

	return writeSummary(w, results)
}

// escapeCell は、Markdown 表のセルを壊す `|` と改行を置き換えます。
func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")

	return strings.ReplaceAll(s, "\n", " ")
}
