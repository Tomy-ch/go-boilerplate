package architest

// Realtime Delivery の計装規約（docs/design/realtime-delivery.md §3.4）を機械的に検査する。
//
//  1. instrument 名はすべて `realtime.` で始まり、リテラルで書かれている（feature 中立）。
//  2. label のキーは宣言した 4 つだけで、主体の識別子（subject / stream / event / …）を label にしない。
//  3. `realtime` meter を取るのは検査対象の 1 ファイルだけ（走査の外へ計装が逃げない）。
//
// 規約 2 が本体。ID を label にすると時系列の本数が使う側の都合で際限なく増え、いったん出した
// 系列は取り消せない。lint も coverage もこれを見ないので、ここで止める。
//
// 突き合わせる 2 つの集合は「ソースが宣言している名前 / キー」と「規約が許す名前 / キー」。
// ソース → 規約の向きが規約違反を捕まえ、逆向き（NotEmpty の canary）は走査器が壊れて
// 何も拾わなくなったときに気づくためにある。規約 3 は、その走査対象そのものが逃げるのを止める。
//
// 残る限界: 走査はテキストなので（depguard が `go/ast` を禁じている）、属性の集合を別の場所で
// 組んでから渡す形（`metric.WithAttributeSet(externallyBuiltSet)`）には到達できない。
// キーを直接書く形はすべて拾うが、この 1 形だけは人手のレビューが受け持つ。

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgfs "go-boilerplate/pkg/fs"
)

// realtimeMetricsFile は、Realtime Delivery の計装が宣言されている唯一のファイル。
const realtimeMetricsFile = "internal/observability/realtime_metrics.go"

// realtimeMeterNameLiteral は、Realtime Delivery の meter 名。これを取れるのは上のファイルだけ。
const realtimeMeterNameLiteral = `"go-boilerplate/realtime"`

// realtimeMetricPrefix は、instrument 名に要求する接頭辞。
const realtimeMetricPrefix = "realtime."

// realtimeAllowedLabelKeys は、label に使ってよいキー。いずれも取り得る値が有界。
// ここを増やす変更は設計正本 §3.4 の書き換えを伴うべきなので、人手の確認を通すために宣言で持つ。
var realtimeAllowedLabelKeys = []string{"outcome", "reason", "result", "trigger"}

// instrumentCallRe は、meterBuilder の生成呼び出しから第 1 引数をそのまま捕捉する。
// レシーバ名を固定しないのは、`b` 以外の名前に変えるだけで検査を外せてしまうため。
var instrumentCallRe = regexp.MustCompile(
	`\.(?:counter|gauge|histogram|countHistogram|upDownCounter)\(\s*([^,]+),`,
)

// rawInstrumentCallRe は、meterBuilder を迂回して OTel の生成子を直接呼ぶ形を捕捉する。
// 迂回されると instrumentCallRe が何も拾わず、名前の検査が丸ごと空振りする。
var rawInstrumentCallRe = regexp.MustCompile(`\.(?:Int64|Float64)(?:Counter|Histogram|Gauge|UpDownCounter)\(`)

// attributeCallRe は、label のキーを与える呼び出しの第 1 引数を捕捉する。
// `String` だけに絞ると、同じ package の既存作法である `attribute.Int` / リテラル引数が素通りする。
var attributeCallRe = regexp.MustCompile(
	`attribute\.(?:String|Int|Int64|Float64|Bool|StringSlice|IntSlice|Stringer|Key)\(\s*([^,)]+)[,)]`,
)

// attributeStructKeyRe は、`attribute.KeyValue{Key: …}` の形でキーを与える書き方を捕捉する。
var attributeStructKeyRe = regexp.MustCompile(`attribute\.KeyValue\{\s*Key:\s*([^,}]+)`)

// labelConstRe は、`attrXxx = "key"` 形式の定数宣言を捕捉する。
var labelConstRe = regexp.MustCompile(`(?m)^\s*(?:const\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"([^"]+)"`)

// stringLiteralRe は、引数がそのまま文字列リテラルである形を捕捉する。
var stringLiteralRe = regexp.MustCompile(`^"([^"]*)"$`)

// idLikeLabelRe は、主体の識別子とみなす label のキー。
var idLikeLabelRe = regexp.MustCompile(
	`(?i)(^id$|_id$|^id_|stream|subject|destination|ticket|trace|span|event|message|user|instance|connection)`,
)

func TestRealtimeMetricNamesAreFeatureNeutral(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("instrument 名はすべてリテラルで realtime. から始まる", func(t *testing.T) {
			t.Parallel()

			src := readRealtimeMetricsSource(t)
			names, violations := realtimeInstrumentNames(src)
			// 計装は sample ではないので、sample 削除後もこの集合は空にならない。
			require.NotEmpty(t, names, "instrument を 1 つも拾えていない。走査の対象かパターンを疑う")

			for _, name := range names {
				if !strings.HasPrefix(name, realtimeMetricPrefix) {
					violations = append(violations, "接頭辞が違う instrument 名: "+name)
				}
			}
			slices.Sort(violations)

			assert.Emptyf(t, violations,
				"%s の instrument 名が規約（%q 始まりのリテラル）に反している（設計正本 §3.4）",
				realtimeMetricsFile, realtimeMetricPrefix)
		})

		t.Run("meterBuilder を迂回して instrument を作っていない", func(t *testing.T) {
			t.Parallel()

			// 迂回されると instrumentCallRe が空を返し、名前の検査が静かに空振りする。
			assert.NotRegexp(t, rawInstrumentCallRe, readRealtimeMetricsSource(t),
				"OTel の生成子を直接呼ぶと名前の検査を通らない。meterBuilder 経由で宣言すること")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接頭辞違反・非リテラル・連結を走査器が拾う", func(t *testing.T) {
			t.Parallel()

			names, violations := realtimeInstrumentNames(`
func f() {
	b.counter("inquiry.messages", "件数")
	b.countHistogram(name, "変数で渡す")
	mb.gauge("realtime."+feature, "連結で接頭辞だけ合わせる")
}`)

			assert.Equal(t, []string{"inquiry.messages"}, names)
			assert.Len(t, violations, 2, "非リテラルと連結の 2 件を違反として拾うこと")
		})
	})
}

func TestRealtimeMetricLabelsCarryNoIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("label のキーは宣言した一覧に収まり識別子を含まない", func(t *testing.T) {
			t.Parallel()

			keys, violations := realtimeLabelKeys(readRealtimeMetricsSource(t))
			require.NotEmpty(t, keys, "label を 1 つも拾えていない。走査の対象かパターンを疑う")

			for _, key := range keys {
				if !slices.Contains(realtimeAllowedLabelKeys, key) {
					violations = append(violations, "未宣言の label キー: "+key)
				}

				if idLikeLabelRe.MatchString(key) {
					violations = append(violations, "識別子に見える label キー: "+key)
				}
			}
			slices.Sort(violations)

			assert.Emptyf(t, violations,
				"%s が ID を label にしている。増やすなら設計正本 §3.4 を先に直すこと", realtimeMetricsFile)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リテラル・定数・String 以外・Key 形式のいずれでも識別子を拾う", func(t *testing.T) {
			t.Parallel()

			// 走査器が素通りしていないことを、違反を含む合成ソースで確かめる。
			// これが無いと、パターンが壊れて何も拾わなくなっても正常系は緑のまま通る。
			// 4 形すべてを置くのは、同じ package の既存作法（リテラル引数）が最も踏みやすいため。
			keys, violations := realtimeLabelKeys(`
const attrSubject = "subject"
func f() {
	a := attribute.String("user_id", id)
	b := attribute.Int("stream_seq", n)
	c := attribute.String(attrSubject, s)
	d := attribute.KeyValue{Key: "ticket_hash", Value: v}
}`)

			assert.Empty(t, violations)
			assert.ElementsMatch(t, []string{"user_id", "stream_seq", "subject", "ticket_hash"}, keys)
			for _, key := range keys {
				assert.Regexp(t, idLikeLabelRe, key, "識別子とみなす判定が %q に働いていない", key)
			}
		})

		t.Run("解決できない label キーは違反として報告される", func(t *testing.T) {
			t.Parallel()

			// 変数でキーを渡されると実値が分からない。素通りさせず違反にする。
			keys, violations := realtimeLabelKeys(`func f() { attribute.String(key, v) }`)

			assert.Empty(t, keys)
			assert.Len(t, violations, 1)
		})
	})
}

func TestRealtimeMeterIsDeclaredInOnePlace(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("realtime meter を取るのは検査対象のファイルだけ", func(t *testing.T) {
			t.Parallel()

			// meter 名は文字列なので、別ファイルから同じ meter を取れば走査の外で
			// realtime.* の系列を増やせてしまう。対象が 1 ファイルであること自体を固定する。
			root := moduleRoot(t)
			matches, err := pkgfs.OS{}.Glob(filepath.Join(root, "internal", "observability", "*.go"))
			require.NoError(t, err)
			require.NotEmpty(t, matches)

			var holders []string
			for _, path := range matches {
				if strings.HasSuffix(path, "_test.go") {
					continue
				}

				b, err := pkgfs.OS{}.ReadFile(path)
				require.NoError(t, err)

				if strings.Contains(string(b), realtimeMeterNameLiteral) {
					rel, err := filepath.Rel(root, path)
					require.NoError(t, err)
					holders = append(holders, rel)
				}
			}
			slices.Sort(holders)

			assert.Equal(t, []string{realtimeMetricsFile}, holders,
				"realtime meter の宣言が増えると、そのファイルの label は検査されない")
		})
	})
}

// readRealtimeMetricsSource は、検査対象のソースを読み出します。
func readRealtimeMetricsSource(t *testing.T) string {
	t.Helper()

	b, err := pkgfs.OS{}.ReadFile(filepath.Join(moduleRoot(t), realtimeMetricsFile))
	require.NoErrorf(t, err, "%s が読めない。ファイルを移動したら宣言も直すこと", realtimeMetricsFile)

	return string(b)
}

// realtimeInstrumentNames は、ソースから instrument 名を宣言順に拾い、
// 名前を静的に読み取れなかった呼び出しを違反として返します。
func realtimeInstrumentNames(src string) ([]string, []string) {
	var names, violations []string

	for _, m := range instrumentCallRe.FindAllStringSubmatch(src, -1) {
		arg := strings.TrimSpace(m[1])

		lit := stringLiteralRe.FindStringSubmatch(arg)
		if lit == nil {
			violations = append(violations, "instrument 名がリテラルでない: "+arg)

			continue
		}

		names = append(names, lit[1])
	}

	return names, violations
}

// realtimeLabelKeys は、ソースから label のキーの実値を重複なく拾い、
// 実値を静的に読み取れなかった呼び出しを違反として返します。
// 引数はリテラルか、このソース内で宣言された定数のどちらかでなければなりません。
func realtimeLabelKeys(src string) ([]string, []string) {
	var keys, violations []string

	values := map[string]string{}
	for _, m := range labelConstRe.FindAllStringSubmatch(src, -1) {
		values[m[1]] = m[2]
	}

	var args []string
	for _, re := range []*regexp.Regexp{attributeCallRe, attributeStructKeyRe} {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			args = append(args, strings.TrimSpace(m[1]))
		}
	}

	for _, arg := range args {
		key, ok := resolveLabelKey(arg, values)
		if !ok {
			violations = append(violations, "label キーの実値を読み取れない: "+arg)

			continue
		}

		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}

	return keys, violations
}

// resolveLabelKey は、引数の字面を label キーの実値に解決します。
func resolveLabelKey(arg string, consts map[string]string) (string, bool) {
	if lit := stringLiteralRe.FindStringSubmatch(arg); lit != nil {
		return lit[1], true
	}

	value, ok := consts[arg]

	return value, ok
}
