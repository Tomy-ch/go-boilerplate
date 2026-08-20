// Package main は、追跡対象の graphify 成果物にマシン固有の情報が残っていないかを検査するツール。
//
//	-dir:  検査する graphify の出力ディレクトリ（既定 graphify-out）
//	-pin:  抽出プロンプトの pin ファイル（既定 .agents/graphify/spec-pin.toml）
//	-spec: これから抽出に使う extraction-spec.md。与えると pin と一致するか照合する
//	-base: この ref からの差分に出力ディレクトリが含まれていないか検査する
//
// graphify の出力ディレクトリには、生成したマシンでしか意味を持たないファイルが同居する。
// 抽出キャッシュの AST 側はノード ID を渡されたパスから組み立てるため絶対パスを焼き込み、
// `.graphify_root` / `.graphify_python` はホストのパスそのものを保持する。相対パスへ直す
// 後処理は最終グラフにしか効かないので、キャッシュ側は絶対パスを抱えたまま残る。これらが
// 追跡対象に入ると、他の checkout では意味を成さない差分が出続けるうえ、公開リポジトリに
// マシンの利用者名が残る。編集で消せない公開でもあるため、混入は入る前に止める。
//
// 検査は 4 つ。
//
//  1. 追跡してよい成果物の一覧から外れたものが追跡されていないか。.gitignore のホワイトリスト
//     が破れた、あるいは意図しない add が起きた場合に当たる。
//  2. 追跡してよい成果物自身にホストの絶対パスが混ざっていないか。上流が新しい経路で漏らし
//     始めた場合に当たる。1 だけでは回帰を捉えられず、2 だけでは絶対パスを含まないマシン
//     ローカルなファイルを見逃すので、両方が要る。
//  3. セマンティックキャッシュの名前空間が pin と一致するか。名前空間は抽出プロンプトの
//     fingerprint で決まるが、そのプロンプトはリポジトリ管理外のファイルなので、キャッシュ
//     だけをコミットすると素性が追えなくなる。-spec を与えれば、これから使うプロンプトが
//     pin と同じものかも同時に照合する。
//  4. -base を与えたとき、その ref からの差分に出力ディレクトリが含まれていないか。グラフは
//     1 ファイル 45 万行の塊で、コードを 1 行触るだけで全面が書き換わるため、並行するブランチ
//     が同時に更新すると必ず衝突する。しかも生成物の衝突を再生成で解くという通常の逃げ道は、
//     意味論抽出にモデルが要る以上ここでは使えない。だから更新はデフォルトブランチ側に寄せ、
//     feature ブランチからは持ち込ませない。発火させる条件は呼び出し側が決める。
//
// このツールは lefthook と CI のゲートから呼ばれる。壊れ方が「何も検査しなくなる」方向に
// 出るため、判定ロジックはシェルの中ではなくテストの当たる Go 側に置く。
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"go-boilerplate/pkg/xerrors"
)

const (
	// defaultDir は graphify の既定の出力ディレクトリ。上流の既定値（GRAPHIFY_OUT 未設定時）
	// と揃えてある。
	defaultDir = "graphify-out"
	// defaultPin は抽出プロンプトの pin ファイル。
	defaultPin = ".agents/graphify/spec-pin.toml"
	// semanticCache は出力ディレクトリからの、セマンティックキャッシュの相対位置。
	semanticCache = "cache/semantic"
	// excerptRadius は違反箇所の前後に何バイト添えて報告するか。graph.json は minify された
	// 1 行 15MB の JSON なので、行番号ではなく抜粋でしか位置を伝えられない。
	excerptRadius = 40
	// fingerprintLength は fingerprint の hex 桁数（graphify の _PROMPT_FP_LEN と同じ）。
	fingerprintLength = 12
	// maxListedChanges は差分の列挙上限。グラフ更新は 8 ファイル前後だが、全件出すと
	// 肝心の「どう戻すか」が流れる。
	maxListedChanges = 10
)

// artifacts は graphify の出力直下で追跡してよいファイル。どの checkout で開いても同じ意味を
// 持つものだけを挙げる。.gitignore の graphify 節は同じ一覧をホワイトリストとして書いており、
// 片方だけを増やすと本ツールが追跡対象外として検出する。
var artifacts = []string{
	"GRAPH_REPORT.md",
	"cost.json",
	"edges.json",
	"graph.html",
	"graph.json",
	"manifest.json",
	"metadata.json",
	"nodes.json",
}

// homePrefixes は成果物を生成したマシンのホームディレクトリを表す接頭辞。大文字の `/Users/` を
// 見るので、この API が持つ `/users` のような小文字のパスとは衝突しない。
var homePrefixes = []string{"/Users/", "/home/", "/root/", `C:\Users\`}

// pinLine は pin ファイルの `key = "value"` 行。lockfile 系スクリプトと同じ手読みで、TOML
// ライブラリは持ち込まない。
var pinLine = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"([^"]*)"$`)

var (
	// errUnexpectedTracked は、追跡してよい成果物の一覧に無いものが追跡されていることを表す。
	errUnexpectedTracked = xerrors.New("unexpected tracked file under graphify output")
	// errMachineLocalPath は、成果物がマシン固有の絶対パスを含むことを表す。
	errMachineLocalPath = xerrors.New("machine-local absolute path in graphify artifact")
	// errSpecPinMismatch は、抽出プロンプトの fingerprint が pin と食い違うことを表す。
	errSpecPinMismatch = xerrors.New("extraction prompt fingerprint differs from the pin")
	// errPinInvalidLine は、pin ファイルに代入として解釈できない行があることを表す。
	errPinInvalidLine = xerrors.New("invalid line in the spec pin")
	// errPinDuplicateKey は、pin ファイルで同じキーが二度定義されていることを表す。
	errPinDuplicateKey = xerrors.New("duplicate key in the spec pin")
	// errPinMissingKey は、pin ファイルに必須キーが無いことを表す。
	errPinMissingKey = xerrors.New("missing key in the spec pin")
	// errArtifactInDiff は、base からの差分に graphify の出力が含まれていることを表す。
	errArtifactInDiff = xerrors.New("graphify output changed relative to the base branch")
)

// violation は 1 件の検出結果。kind は判定の種類、detail は人が読む説明。
type violation struct {
	path   string
	kind   error
	detail string
}

// hit は成果物 1 ファイル内で見つかった絶対パスの 1 件。
type hit struct {
	prefix  string
	offset  int
	excerpt string
}

// main は 1:1 テスト規約の対象外で分岐を検査できないため、判断は run に置きます。
func main() {
	log.SetFlags(0)

	if err := run(os.Args[1:], listTracked, os.ReadFile, diffAgainst); err != nil {
		log.Fatalf("%v", err)
	}
}

// run は、追跡中の graphify 成果物を検査して違反を報告します。
// list は追跡パスの列挙手段、read はファイルの読み出し手段、diff は base ref からの変更パスの
// 列挙手段で、いずれも差し替えられるよう引数で受けます。
func run(
	args []string,
	list func(dir string) ([]string, error),
	read func(name string) ([]byte, error),
	diff func(base, dir string) ([]string, error),
) error {
	fs := flag.NewFlagSet("graphify-check", flag.ContinueOnError)
	dir := fs.String("dir", defaultDir, "検査する graphify の出力ディレクトリ")
	pin := fs.String("pin", defaultPin, "抽出プロンプトの pin ファイル")
	spec := fs.String("spec", "", "これから抽出に使う extraction-spec.md（省略時は照合しない）")
	base := fs.String("base", "", "この ref からの差分に出力が含まれていないか検査する（省略時は検査しない）")

	if err := fs.Parse(args); err != nil {
		// ヘルプ要求は失敗ではないので 0 で終える。usage は flag が既に出力している。
		if xerrors.Is(err, flag.ErrHelp) {
			return nil
		}

		return xerrors.Wrap(err, "failed to parse flags")
	}

	fingerprint, err := readPinnedFingerprint(*pin, read)
	if err != nil {
		return err
	}

	if *spec != "" {
		if err := verifySpec(*spec, fingerprint, read); err != nil {
			return err
		}
	}

	if *base != "" {
		if err := verifyBase(*base, *dir, diff); err != nil {
			return err
		}
	}

	tracked, err := list(*dir)
	if err != nil {
		return xerrors.Wrap(err, "❌ 追跡ファイルを列挙できません")
	}
	// 追跡ゼロは「何も見なかった」ことであって「違反が無かった」ことではないので、合格と
	// 同じ文言では報告しない。ただし追跡が無ければコミットされる成果物も無く、守るべきものが
	// 無いため失敗にはしない。graphify を回す前の checkout や、出力を丸ごと無視する運用が
	// これに当たる。壊れた走査（git が答えない・成果物が読めない）は下で失敗になる。
	if len(tracked) == 0 {
		log.Printf("graphify: %s に追跡中の成果物がないため、検査対象はありません", *dir)

		return nil
	}

	violations, err := inspect(*dir, fingerprint, tracked, read)
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		log.Printf("✅ graphify: %d ファイルを検査、マシン固有の情報はありません", len(tracked))

		return nil
	}

	// 違反の中身は行ごとに出し、戻り値には種類だけを載せる。呼び出し側が分岐に使うのは種類で、
	// どのファイルがどう汚れているかは人が読むログの側の情報である。
	kinds := make([]error, 0, len(violations))
	for _, found := range violations {
		log.Printf("❌ %s: %s", found.path, found.detail)
		if !slices.Contains(kinds, found.kind) {
			kinds = append(kinds, found.kind)
		}
	}

	return xerrors.Join(kinds...)
}

// verifySpec は、これから抽出に使うプロンプトが pin と同じものかを照合します。
// 素性の違うプロンプトで焼いたキャッシュは、名前空間が変わって全件ミスになるか、あるいは
// 名前空間を偽って別 vintage の結果を再生し続けるかのどちらかになります。
func verifySpec(path, pinned string, read func(name string) ([]byte, error)) error {
	body, err := read(path)
	if err != nil {
		return xerrors.Wrap(err, "❌ "+path+" を読み込めません")
	}

	actual := promptFingerprint(body)
	if actual != pinned {
		return xerrors.Wrap(errSpecPinMismatch, fmt.Sprintf(
			"❌ %s の fingerprint は %s で、pin の %s と一致しません"+
				"（別バージョンの graphify、または別エージェント向けの spec を掴んでいます）",
			path, actual, pinned,
		))
	}

	log.Printf("✅ graphify: 抽出プロンプトは pin と一致します (%s)", pinned)

	return nil
}

// verifyBase は、base ref からの差分に graphify の出力が含まれていないかを見ます。
func verifyBase(base, dir string, diff func(base, dir string) ([]string, error)) error {
	changed, err := diff(base, dir)
	if err != nil {
		return xerrors.Wrap(err, "❌ "+base+" からの差分を取得できません")
	}
	if len(changed) == 0 {
		return nil
	}

	log.Printf("❌ %s からの差分に graphify の出力が %d 件含まれています:", base, len(changed))
	for _, path := range changed[:min(len(changed), maxListedChanges)] {
		log.Printf("     %s", path)
	}
	if len(changed) > maxListedChanges {
		log.Printf("     ... 他 %d 件", len(changed)-maxListedChanges)
	}

	return xerrors.Wrap(errArtifactInDiff, fmt.Sprintf(
		"❌ %s の更新はデフォルトブランチ側に寄せています。このブランチからは外してください"+
			"（`git checkout %s -- %s` で戻せます）", dir, base, dir,
	))
}

// diffAgainst は、base ref と作業ツリーの間で dir 配下に変更のあったパスを返します。
// 三点ではなく二点の差分を使います。三点の分岐点は base が進むたびに動くので、同じブランチが
// 触ってもいない出力を「変更した」と報告し得ます。
func diffAgainst(base, dir string) ([]string, error) {
	output, err := exec.CommandContext( //nolint:gosec // 引数は base と dir のみで、コマンド自体は固定
		context.Background(), "git", "diff", "--name-only", "-z", base, "--", dir,
	).Output()
	if err != nil {
		return nil, xerrors.Wrap(err, "git diff "+base+" -- "+dir)
	}

	var changed []string
	for name := range strings.SplitSeq(string(output), "\x00") {
		if name != "" {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)

	return changed, nil
}

// readPinnedFingerprint は、pin ファイルから抽出プロンプトの fingerprint を読みます。
// 読み飛ばしや後勝ちの上書きは、その宣言が「存在しない」あるいは「行順で決まる」状態を
// 警告なく作るため、解釈できない行と重複キーはエラーにします。
func readPinnedFingerprint(path string, read func(name string) ([]byte, error)) (string, error) {
	body, err := read(path)
	if err != nil {
		return "", xerrors.Wrap(err, "❌ "+path+" を読み込めません")
	}

	values := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matched := pinLine.FindStringSubmatch(line)
		if matched == nil {
			return "", xerrors.Wrap(errPinInvalidLine, fmt.Sprintf("%s の %d 行目: %q", path, lineNo, line))
		}
		if _, duplicate := values[matched[1]]; duplicate {
			return "", xerrors.Wrap(errPinDuplicateKey, fmt.Sprintf("%s の %d 行目: %q", path, lineNo, matched[1]))
		}
		values[matched[1]] = matched[2]
	}
	if err := scanner.Err(); err != nil {
		return "", xerrors.Wrap(err, "❌ "+path+" を読めません")
	}

	fingerprint := values["spec_fingerprint"]
	if fingerprint == "" {
		return "", xerrors.Wrap(errPinMissingKey, path+" に spec_fingerprint がありません")
	}

	return fingerprint, nil
}

// promptFingerprint は、抽出プロンプトの fingerprint を graphify.cache.prompt_fingerprint と
// 同じ規則で求めます。CRLF/CR を LF へ正規化し、各行の行末空白を落とし、前後の空白を落として
// から SHA-256 の先頭 12 桁を採ります。改行コードの正規化は、CRLF で checkout した Windows が
// LF の checkout と別の fingerprint を出して全件再抽出になるのを防ぐためのものです。
func promptFingerprint(body []byte) string {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRightFunc(line, unicode.IsSpace)
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.Join(lines, "\n"))))

	return hex.EncodeToString(sum[:])[:fingerprintLength]
}

// listTracked は、dir 配下で git が追跡しているパスを昇順で返します。
// index を読むので、まだコミットされていない staged なファイルも対象に入ります。
func listTracked(dir string) ([]string, error) {
	output, err := exec.CommandContext(context.Background(), "git", "ls-files", "-z", "--", dir).Output() //nolint:gosec // 引数は dir のみで、コマンド自体は固定
	if err != nil {
		return nil, xerrors.Wrap(err, "git ls-files "+dir)
	}

	var tracked []string
	// -z 区切りなので、パスに改行を含むファイルでも分割が壊れない。
	for name := range strings.SplitSeq(string(output), "\x00") {
		if name != "" {
			tracked = append(tracked, name)
		}
	}
	sort.Strings(tracked)

	return tracked, nil
}

// inspect は、追跡パスをホワイトリスト・絶対パス混入・キャッシュ名前空間の 3 観点で検査します。
func inspect(
	dir, fingerprint string, tracked []string, read func(name string) ([]byte, error),
) ([]violation, error) {
	var violations []violation
	// 名前空間は個々のエントリではなくディレクトリの性質なので、件数だけ数えて後でまとめて
	// 判定する。エントリ 1 件ごとに報告すると、同じ文言が数百行並んで肝心の名前が埋もれる。
	namespaces := map[string]int{}
	for _, file := range tracked {
		name, under := strings.CutPrefix(file, dir+"/")
		if !under {
			violations = append(violations, unexpected(file))

			continue
		}

		if entry, cached := strings.CutPrefix(name, semanticCache+"/"); cached {
			namespace, _, nested := strings.Cut(entry, "/")
			if !nested {
				namespace = ""
			}
			namespaces[namespace]++

			continue
		}

		if !slices.Contains(artifacts, name) {
			violations = append(violations, unexpected(file))

			continue
		}

		body, err := read(file)
		if err != nil {
			return nil, xerrors.Wrap(err, "❌ "+file+" を読み込めません")
		}
		for _, found := range scanHomePrefixes(body) {
			violations = append(violations, violation{
				path: file,
				kind: errMachineLocalPath,
				detail: fmt.Sprintf(
					"マシン固有の絶対パス %q を含みます (offset %d: %s)", found.prefix, found.offset, found.excerpt,
				),
			})
		}
	}

	return append(violations, checkNamespaces(dir, fingerprint, namespaces)...), nil
}

// checkNamespaces は、追跡されているセマンティックキャッシュの名前空間が pin と一致するかを
// 見ます。namespaces は名前空間ごとのエントリ数で、空文字は名前空間を持たないフラット配置です。
//
// フラット配置は通しません。graphify はそこを「fingerprint 導入前の、素性が不明なエントリ」の
// 置き場として扱い、どのプロンプトからでもヒットさせたうえで警告を出します（upstream #1939）。
// 共有するキャッシュの種としては、その素性不明が恒久的に残ってしまいます。
func checkNamespaces(dir, fingerprint string, namespaces map[string]int) []violation {
	expected := "p" + fingerprint

	var violations []violation
	for _, namespace := range slices.Sorted(maps.Keys(namespaces)) {
		if namespace == expected {
			continue
		}

		path := dir + "/" + semanticCache + "/" + namespace
		detail := fmt.Sprintf(
			"%d 件のエントリの名前空間が pin の %s と一致しません"+
				"（別の抽出プロンプトで焼いたキャッシュです。焼き直すか %s を更新してください）",
			namespaces[namespace], expected, defaultPin,
		)
		if namespace == "" {
			path = dir + "/" + semanticCache
			detail = fmt.Sprintf(
				"%d 件のエントリが名前空間を持たないフラット配置です"+
					"（素性不明のエントリとして扱われるため、%s/ の下へ置いてください）",
				namespaces[""], expected,
			)
		}
		violations = append(violations, violation{path: path, kind: errSpecPinMismatch, detail: detail})
	}

	return violations
}

// unexpected は、追跡してよい一覧から外れたファイルの違反を作ります。
func unexpected(file string) violation {
	return violation{
		path: file,
		kind: errUnexpectedTracked,
		detail: "追跡対象外です。マシンローカルな派生物が紛れていないか " +
			".gitignore の graphify 節を確認してください",
	}
}

// scanHomePrefixes は、body から絶対パスを探して接頭辞ごとに最初の 1 件だけを位置順で返します。
// 混入は 1 ファイルあたり数千件に及ぶため、全件を報告すると「どのファイルが汚染されているか」
// という肝心の情報がログに埋もれます。
func scanHomePrefixes(body []byte) []hit {
	var hits []hit
	for _, prefix := range homePrefixes {
		at := bytes.Index(body, []byte(prefix))
		if at < 0 {
			continue
		}
		hits = append(hits, hit{prefix: prefix, offset: at, excerpt: excerpt(body, at)})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].offset < hits[j].offset })

	return hits
}

// excerpt は、body の offset 周辺を引用符付きの 1 行に畳んで返します。
func excerpt(body []byte, at int) string {
	start := max(at-excerptRadius, 0)
	end := min(at+excerptRadius, len(body))

	return strconv.Quote(string(body[start:end]))
}
