package main

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

// fakeAPIError は、SDK のサービスエラーと同じ形（ErrorCode / ErrorMessage）を持つテスト用エラーです。
type fakeAPIError struct {
	code string
	msg  string
}

// fakeNetError は、応答前に失敗した transport エラーです。
type fakeNetError struct{}

func (e *fakeAPIError) Error() string        { return e.code + ": " + e.msg }
func (e *fakeAPIError) ErrorCode() string    { return e.code }
func (e *fakeAPIError) ErrorMessage() string { return e.msg }

func (fakeNetError) Error() string   { return "dial tcp: connection refused" }
func (fakeNetError) Timeout() bool   { return false }
func (fakeNetError) Temporary() bool { return false }

func Test_classify(t *testing.T) {
	t.Parallel()

	var _ net.Error = fakeNetError{}

	tests := map[string]struct {
		err  error
		want Verdict
	}{
		"nil は互換":                            {err: nil, want: VerdictCompatible},
		"事後条件の不成立は非互換":                       {err: incompatible("envelope が届いた"), want: VerdictIncompatible},
		"context の期限切れは検証不能":                 {err: xerrors.Wrap(context.DeadlineExceeded, "receive"), want: VerdictUnverifiable},
		"net.Error は検証不能":                    {err: xerrors.Wrap(fakeNetError{}, "dial"), want: VerdictUnverifiable},
		"InvalidAction は未対応":                 {err: &fakeAPIError{code: "InvalidAction", msg: "no"}, want: VerdictUnsupported},
		"NotImplemented は未対応":                {err: &fakeAPIError{code: "NotImplemented", msg: "no"}, want: VerdictUnsupported},
		"deserialization failed（XML 応答）は未対応": {err: xerrors.New("operation error SQS: ListQueues, deserialization failed"), want: VerdictUnsupported},
		"404 は未対応":                           {err: xerrors.New("https response error StatusCode: 404, RequestID: x"), want: VerdictUnsupported},
		"ConditionalCheckFailedException は非互換（受理された拒否）": {
			err:  &fakeAPIError{code: "ConditionalCheckFailedException", msg: "x"},
			want: VerdictIncompatible,
		},
		"分類できない応答は非互換": {err: xerrors.New("something else"), want: VerdictIncompatible},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, classify(tt.err))
		})
	}
}

func Test_isTransportFailure(t *testing.T) {
	t.Parallel()

	assert.True(t, isTransportFailure(context.DeadlineExceeded))
	assert.True(t, isTransportFailure(xerrors.Wrap(context.Canceled, "x")))
	assert.True(t, isTransportFailure(fakeNetError{}))
	assert.False(t, isTransportFailure(&fakeAPIError{code: "InvalidAction", msg: "x"}))
	assert.False(t, isTransportFailure(xerrors.New("plain")))
}

func Test_hasUnsupportedMarker(t *testing.T) {
	t.Parallel()

	assert.True(t, hasUnsupportedMarker("Operation Not Implemented"), "大文字小文字を無視する")
	assert.True(t, hasUnsupportedMarker("https response error StatusCode: 501"))
	assert.False(t, hasUnsupportedMarker("StatusCode: 400, InvalidParameterValue"))
	assert.False(t, hasUnsupportedMarker(""))
}

func Test_describe(t *testing.T) {
	t.Parallel()

	t.Run("API エラーはコードとメッセージに要約する", func(t *testing.T) {
		t.Parallel()

		got := describe(xerrors.Wrap(&fakeAPIError{code: "InvalidAction", msg: "not\nsupported"}, "call"))
		assert.Equal(t, "InvalidAction: not supported", got)
	})

	t.Run("上限を超える本文は切り詰める", func(t *testing.T) {
		t.Parallel()

		got := describe(xerrors.New(strings.Repeat("x", maxDetailLen+10)))
		assert.Len(t, got, maxDetailLen+len("…"))
		assert.True(t, strings.HasSuffix(got, "…"))
	})

	t.Run("上限ちょうどは切り詰めない", func(t *testing.T) {
		t.Parallel()

		got := describe(xerrors.New(strings.Repeat("x", maxDetailLen)))
		assert.Len(t, got, maxDetailLen)
	})
}

func Test_incompatible(t *testing.T) {
	t.Parallel()

	err := incompatible("detail")

	var inc *incompatibleError
	require.ErrorAs(t, err, &inc)
	assert.Equal(t, VerdictIncompatible, classify(err))
}

func Test_incompatibleError_Error(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "body が違う", (&incompatibleError{detail: "body が違う"}).Error())
}

func Test_recorder_add(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	rec.add("D1", "s", "c", VerdictCompatible, "d")
	rec.add("D2", "s", "c2", VerdictUnsupported, "d2")

	require.Len(t, rec.results, 2)
	assert.Equal(t, Result{ID: "D2", Subject: "s", Check: "c2", Verdict: VerdictUnsupported, Detail: "d2"}, rec.results[1])
}

func Test_recorder_record(t *testing.T) {
	t.Parallel()

	t.Run("成功は渡された detail をそのまま載せる", func(t *testing.T) {
		t.Parallel()

		rec := &recorder{}
		got := rec.record("D1", "s", "c", "ok detail", nil)
		assert.Equal(t, VerdictCompatible, got)
		assert.Equal(t, "ok detail", rec.results[0].Detail)
	})

	t.Run("失敗はエラーの要約で detail を置き換える", func(t *testing.T) {
		t.Parallel()

		rec := &recorder{}
		got := rec.record("D1", "s", "c", "ignored", &fakeAPIError{code: "InvalidAction", msg: "no"})
		assert.Equal(t, VerdictUnsupported, got)
		assert.Equal(t, "InvalidAction: no", rec.results[0].Detail)
	})
}

func Test_recorder_skip(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	rec.skip("G9", "s", "c", "先行検査 G1 により実行不能")

	require.Len(t, rec.results, 1)
	assert.Equal(t, VerdictUnverifiable, rec.results[0].Verdict)
	assert.Equal(t, "先行検査 G1 により実行不能", rec.results[0].Detail)
}

func Test_runChain(t *testing.T) {
	t.Parallel()

	pass := func(context.Context) (string, error) { return "ok", nil }
	fail := func(context.Context) (string, error) { return "", &fakeAPIError{code: "InvalidAction", msg: "no"} }

	t.Run("halt な検査の失敗は後続を検証不能として残す", func(t *testing.T) {
		t.Parallel()

		rec := &recorder{}
		first := runChain(t.Context(), "s", []step{
			{id: "1", check: "a", halt: true, fn: fail},
			{id: "2", check: "b", fn: pass},
			{id: "3", check: "c", fn: pass},
		}, rec)

		require.Len(t, rec.results, 3)
		assert.False(t, first)
		assert.Equal(t, VerdictUnsupported, rec.results[0].Verdict)
		assert.Equal(t, VerdictUnverifiable, rec.results[1].Verdict)
		assert.Equal(t, VerdictUnverifiable, rec.results[2].Verdict)
		assert.Contains(t, rec.results[2].Detail, "先行検査 1")
	})

	t.Run("halt でない検査の失敗は後続を止めない", func(t *testing.T) {
		t.Parallel()

		rec := &recorder{}
		first := runChain(t.Context(), "s", []step{
			{id: "1", check: "a", fn: pass},
			{id: "2", check: "b", fn: fail},
			{id: "3", check: "c", fn: pass},
		}, rec)

		require.Len(t, rec.results, 3)
		assert.True(t, first)
		assert.Equal(t, VerdictCompatible, rec.results[2].Verdict)
		assert.Equal(t, "ok", rec.results[2].Detail)
	})

	t.Run("先頭が非 halt で失敗しても後続は走り、戻り値は false", func(t *testing.T) {
		t.Parallel()

		rec := &recorder{}
		first := runChain(t.Context(), "s", []step{{id: "1", check: "a", fn: fail}, {id: "2", check: "b", fn: pass}}, rec)

		assert.False(t, first)
		assert.Equal(t, VerdictCompatible, rec.results[1].Verdict)
	})
}

func Test_cleanupContext(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithCancel(t.Context())
	cancel()

	ctx, cleanup := cleanupContext(parent)
	defer cleanup()

	require.NoError(t, ctx.Err(), "親の cancel を引き継がない")

	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(cleanupTimeout), deadline, time.Second)
}

func Test_count(t *testing.T) {
	t.Parallel()

	c := count([]Result{{Verdict: VerdictCompatible}, {Verdict: VerdictCompatible}, {Verdict: VerdictUnsupported}})

	assert.Equal(t, 2, c[VerdictCompatible])
	assert.Equal(t, 1, c[VerdictUnsupported])
	assert.Equal(t, 0, c[VerdictIncompatible])
}

func Test_exitCode(t *testing.T) {
	t.Parallel()

	compatible := Result{ID: "1", Verdict: VerdictCompatible}
	incompatibleResult := Result{ID: "2", Verdict: VerdictIncompatible}
	unsupported := Result{ID: "3", Verdict: VerdictUnsupported}
	unverifiable := Result{ID: "4", Verdict: VerdictUnverifiable}

	t.Run("結果が 0 件なら失敗（検査していないことをクリーンにしない）", func(t *testing.T) {
		t.Parallel()

		code, err := exitCode(nil, false)
		require.ErrorIs(t, err, errNoResults)
		assert.Equal(t, 1, code)
	})

	tests := map[string]struct {
		results []Result
		strict  bool
		want    int
	}{
		"全て互換なら 0":             {results: []Result{compatible}, want: 0},
		"検証不能が 1 件でもあれば 1":     {results: []Result{compatible, unverifiable}, want: 1},
		"非互換は strict でなければ 0":  {results: []Result{compatible, incompatibleResult}, want: 0},
		"非互換は strict なら 1":     {results: []Result{compatible, incompatibleResult}, strict: true, want: 1},
		"未対応は strict なら 1":     {results: []Result{unsupported}, strict: true, want: 1},
		"未対応は strict でなければ 0":  {results: []Result{unsupported}, want: 0},
		"strict でも検証不能は変わらず 1": {results: []Result{unverifiable}, strict: true, want: 1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code, err := exitCode(tt.results, tt.strict)
			require.NoError(t, err)
			assert.Equal(t, tt.want, code)
		})
	}
}

func Test_writeMarkdown(t *testing.T) {
	t.Parallel()

	results := []Result{
		{ID: "D1", Subject: "DynamoDB Local", Check: "CreateTable", Verdict: VerdictCompatible, Detail: "table a|b"},
		{ID: "G4", Subject: "GoAWS SNS/SQS", Check: "Policy", Verdict: VerdictIncompatible, Detail: "silent drop"},
		{ID: "G9", Subject: "GoAWS SNS/SQS", Check: "cleanup", Verdict: VerdictUnverifiable, Detail: "skipped"},
	}

	var buf bytes.Buffer
	require.NoError(t, writeMarkdown(&buf, results))
	out := buf.String()

	assert.Contains(t, out, "| # | Subject | Check | Verdict | Detail |")
	assert.Contains(t, out, "| D1 | DynamoDB Local | CreateTable | **互換** | table a\\|b |")
	assert.Contains(t, out, "互換 1 / 非互換 1 / 未対応 0 / 検証不能 1")
	assert.Contains(t, out, "- G4 Policy: 非互換 — silent drop")
	assert.NotContains(t, out, "- G9", "検証不能は compatibility implementation 候補に載せない")
}

func Test_writeSummary(t *testing.T) {
	t.Parallel()

	t.Run("候補が無ければ「なし」", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		require.NoError(t, writeSummary(&buf, []Result{{ID: "D1", Verdict: VerdictCompatible}}))
		assert.Contains(t, buf.String(), "- なし")
	})

	t.Run("未対応も候補に載る", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		require.NoError(t, writeSummary(&buf, []Result{{ID: "G0", Check: "probe", Verdict: VerdictUnsupported, Detail: "404"}}))
		assert.Contains(t, buf.String(), "- G0 probe: 未対応 — 404")
	})
}

func Test_writeText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeText(&buf, []Result{{ID: "D1", Subject: "s", Check: "c", Verdict: VerdictCompatible, Detail: "d"}}))

	assert.True(t, strings.HasPrefix(buf.String(), "D1 "))
	assert.Contains(t, buf.String(), "互換 1 / 非互換 0 / 未対応 0 / 検証不能 0")
}

func Test_escapeCell(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "a\\|b c", escapeCell("a|b\nc"))
	assert.Equal(t, "plain", escapeCell("plain"))
}
