package main

import (
	"bufio"
	"bytes"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goModFixture は require ブロックが 2 つに割れた実際の go.mod の形を最小で再現する。
const goModFixture = `module go-boilerplate

go 1.26.5

require (
	github.com/labstack/echo/v4 v4.15.4
	github.com/BurntSushi/toml v1.4.0
)

require (
	github.com/aws/smithy-go v1.27.6 // indirect
	golang.org/x/sys v0.45.0 // indirect
)
`

// goModOneRequire は、direct な require を 1 本だけ持つ最小の go.mod。
const goModOneRequire = `module m

go 1.26.5

require example.com/mod v1.0.0
`

// roundTripFunc は http.Client の往復を関数で受ける。proxy の宛先は定数なので、
// 実ネットワークへ出さずに応答を与えられるのはトランスポートの層だけである。
type roundTripFunc func(*http.Request) (*http.Response, error)

func writeBypass(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bypass.toml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func day(s string) time.Time {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return d
}

func keysOf(reqs []requirement) []string {
	keys := make([]string, 0, len(reqs))
	for _, r := range reqs {
		keys = append(keys, r.key())
	}
	return keys
}

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func proxyResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func infoBody(published time.Time) string {
	return `{"Version":"v1","Time":"` + published.Format(time.RFC3339) + `"}`
}

// stubProxy は module proxy への往復を差し替える。inspect が http.Client を内部で組み立てる
// ため差し替え口はプロセス共有の DefaultTransport しかなく、呼び出し側は並列化できない。
func stubProxy(t *testing.T, respond func(path string) (int, string)) {
	t.Helper()
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		code, body := respond(r.URL.Path)
		return proxyResponse(code, body), nil
	})
}

// captureLog は log の出力先を差し替える。出力先はプロセス共有なので並列化できない。
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

// stubProxyPublishedAt は、どのバージョンにも同じ公開時刻を返す module proxy を差し込む。
func stubProxyPublishedAt(t *testing.T, published time.Time) {
	t.Helper()
	stubProxy(t, func(string) (int, string) { return http.StatusOK, infoBody(published) })
}

// useWorkTree は go.mod だけを持つ作業ディレクトリを作り、そこへ移動する。パスを返すのは
// バイパス lockfile を同じツリーへ置けるようにするため。
func useWorkTree(t *testing.T, goMod string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	t.Chdir(dir)
	return dir
}

// useGateRepo は base 側 go.mod を 1 コミット持つリポジトリを作り、作業ツリーの go.mod を
// current へ差し替えてそこへ移動する。gate の差分は git 越しに取るため実リポジトリが要る。
func useGateRepo(t *testing.T, base, current string) {
	t.Helper()
	dir := t.TempDir()
	initGoModRepo(t, dir, base)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(current), 0o600))
	t.Chdir(dir)
}

// writeWorkTreeBypass は、作業ディレクトリの規定位置へバイパス lockfile を書く。
func writeWorkTreeBypass(t *testing.T, dir, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(bypassFile)), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, bypassFile), []byte(body), 0o600))
}

// initGoModRepo は go.mod を 1 コミットだけ持つ git リポジトリを作る。ホストの設定に
// 左右されないよう著者と署名を固定する。
func initGoModRepo(t *testing.T, dir, goMod string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	base := []string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=t", "-c", "commit.gpgsign=false"}
	for _, args := range [][]string{{"init", "-q"}, {"add", "go.mod"}, {"commit", "-q", "-m", "init"}} {
		//nolint:gosec // 引数は本ファイル内のリテラルと t.TempDir のパス
		out, err := exec.CommandContext(t.Context(), "git", append(base, args...)...).CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
}

func Test_parseGoMod(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数の require ブロックを跨いで読む", func(t *testing.T) {
			t.Parallel()
			got, err := parseGoMod([]byte(goModFixture))
			require.NoError(t, err)

			keys := make([]string, 0, len(got))
			for _, r := range got {
				keys = append(keys, r.key())
			}
			assert.ElementsMatch(t, []string{
				"github.com/labstack/echo/v4@v4.15.4",
				"github.com/BurntSushi/toml@v1.4.0",
				"github.com/aws/smithy-go@v1.27.6",
				"golang.org/x/sys@v0.45.0",
			}, keys)
		})

		t.Run("indirect 注釈の有無を保持する", func(t *testing.T) {
			t.Parallel()
			got, err := parseGoMod([]byte(goModFixture))
			require.NoError(t, err)

			indirect := map[string]bool{}
			for _, r := range got {
				indirect[r.module] = r.indirect
			}
			assert.False(t, indirect["github.com/labstack/echo/v4"])
			assert.True(t, indirect["github.com/aws/smithy-go"])
		})

		t.Run("ブロックを使わない単一行の require も読む", func(t *testing.T) {
			t.Parallel()
			got, err := parseGoMod([]byte("module m\n\ngo 1.26.5\n\nrequire github.com/x/y v1.2.3\n"))
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "github.com/x/y@v1.2.3", got[0].key())
		})

		t.Run("module / go 行や閉じ括弧を依存として拾わない", func(t *testing.T) {
			t.Parallel()
			got, err := parseGoMod([]byte(goModFixture))
			require.NoError(t, err)
			for _, r := range got {
				assert.NotEqual(t, "module", r.module)
				assert.NotEqual(t, "go", r.module)
			}
		})

		// 行末コメントを落とさないと require 行が正規表現に当たらず、その依存は検査対象から
		// 黙って消える。indirect 注釈だけは意味を持つので、落とす対象から外す。
		t.Run("indirect 以外の行末コメントは落として require を読む", func(t *testing.T) {
			t.Parallel()
			got, err := parseGoMod([]byte("module m\n\nrequire (\n\tgithub.com/x/y v1.2.3 // 一時ピン\n)\n"))
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "github.com/x/y@v1.2.3", got[0].key())
			assert.False(t, got[0].indirect)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// 走査に失敗した go.mod を「依存ゼロ」として返すと、gate は何も見ないまま通る。
		t.Run("1 行が走査上限を超える go.mod はエラーにする", func(t *testing.T) {
			t.Parallel()
			got, err := parseGoMod([]byte("module m\n// " + strings.Repeat("a", bufio.MaxScanTokenSize) + "\n"))
			require.Error(t, err)
			assert.Empty(t, got)
		})
	})
}

func Test_escapePath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("大文字を ! + 小文字へ変換する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "github.com/!burnt!sushi/toml", escapePath("github.com/BurntSushi/toml"))
		})

		t.Run("小文字だけのパスは変えない", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "golang.org/x/sys", escapePath("golang.org/x/sys"))
		})
	})
}

func Test_diffAdded(t *testing.T) {
	t.Parallel()

	before := []requirement{
		{module: "a", version: "v1.0.0"},
		{module: "b", version: "v1.0.0"},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("新規追加のみを返す", func(t *testing.T) {
			t.Parallel()
			got := diffAdded(before, append(before, requirement{module: "c", version: "v0.1.0"}))
			require.Len(t, got, 1)
			assert.Equal(t, "c@v0.1.0", got[0].key())
		})

		t.Run("バージョンを上げたモジュールも対象にする", func(t *testing.T) {
			t.Parallel()
			got := diffAdded(before, []requirement{
				{module: "a", version: "v1.0.0"},
				{module: "b", version: "v2.0.0"},
			})
			require.Len(t, got, 1)
			assert.Equal(t, "b@v2.0.0", got[0].key())
		})

		t.Run("変化が無ければ空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, diffAdded(before, before))
		})
	})
}

func Test_readBypasses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コメントと空行を読み飛ばして 1 エントリを読む", func(t *testing.T) {
			t.Parallel()
			path := writeBypass(t, "# comment\n\n"+
				`"github.com/x/y@v1.2.3" = { expires = 2026-11-06, issue = 931, reason = "CRITICAL への即応" }`+"\n")
			got, err := readBypasses(path)
			require.NoError(t, err)
			require.Len(t, got, 1)

			b := got["github.com/x/y@v1.2.3"]
			assert.Equal(t, day("2026-11-06"), b.expires)
			assert.Equal(t, 931, b.issue)
			assert.Equal(t, "CRITICAL への即応", b.reason)
		})

		t.Run("ファイルが無ければ空として扱う", func(t *testing.T) {
			t.Parallel()
			got, err := readBypasses(filepath.Join(t.TempDir(), "absent.toml"))
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解釈できない行はエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := readBypasses(writeBypass(t, "これは TOML ではない\n"))
			require.Error(t, err)
			assert.ErrorIs(t, err, errBypassInvalidLine)
		})

		t.Run("expires を欠いた行はエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := readBypasses(writeBypass(t,
				`"github.com/x/y@v1.2.3" = { issue = 931, reason = "期限が無い" }`+"\n"))
			require.Error(t, err)
			assert.ErrorIs(t, err, errBypassInvalidLine)
		})

		t.Run("キーの重複はエラーにする", func(t *testing.T) {
			t.Parallel()
			line := `"github.com/x/y@v1.2.3" = { expires = 2026-11-06, issue = 931, reason = "r" }` + "\n"
			_, err := readBypasses(writeBypass(t, line+line))
			require.Error(t, err)
			assert.ErrorIs(t, err, errBypassDuplicateKey)
		})

		// 不在だけを空として扱う。読めない理由を一緒くたに空へ丸めると、バイパスの検査が
		// 「1 件も無い」と報告したまま素通りする。
		t.Run("不在以外の理由で開けないパスはエラーにする", func(t *testing.T) {
			t.Parallel()
			notDir := filepath.Join(t.TempDir(), "not-a-directory")
			require.NoError(t, os.WriteFile(notDir, nil, 0o600))

			_, err := readBypasses(filepath.Join(notDir, "bypass.toml"))
			require.Error(t, err)
			assert.NotErrorIs(t, err, os.ErrNotExist)
		})

		// 書式だけ通って日付として成立しない期限を素通しすると、期限切れ検査が空回りする。
		t.Run("書式は合うが暦日として成立しない expires はエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := readBypasses(writeBypass(t,
				`"github.com/x/y@v1.2.3" = { expires = 2026-13-45, issue = 931, reason = "r" }`+"\n"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "1 行目の expires")
		})

		t.Run("int に収まらない issue 番号はエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := readBypasses(writeBypass(t,
				`"github.com/x/y@v1.2.3" = { expires = 2026-11-06, issue = 99999999999999999999, reason = "r" }`+"\n"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "1 行目の issue")
		})
	})
}

func Test_validateBypasses(t *testing.T) {
	t.Parallel()

	today := day("2026-08-06")
	current := []requirement{{module: "github.com/x/y", version: "v1.2.3"}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限内かつ go.mod にあるエントリは違反にならない", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"github.com/x/y@v1.2.3": {expires: day("2026-09-06"), issue: 931, reason: "r", line: 1},
			}, current, today)
			assert.Empty(t, violations)
			assert.Empty(t, invalid)
		})

		t.Run("期限当日は期限内として扱う", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"github.com/x/y@v1.2.3": {expires: today, issue: 931, reason: "r", line: 1},
			}, current, today)
			assert.Empty(t, violations)
			assert.Empty(t, invalid)
		})

		t.Run("上限ちょうどの期限は許す", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"github.com/x/y@v1.2.3": {expires: today.AddDate(0, maxBypassMonths, 0), issue: 931, reason: "r", line: 1},
			}, current, today)
			assert.Empty(t, violations)
			assert.Empty(t, invalid)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限切れは違反にし効力も失わせる", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"github.com/x/y@v1.2.3": {expires: day("2026-08-05"), issue: 931, reason: "r", line: 3},
			}, current, today)
			require.Len(t, violations, 1)
			assert.Contains(t, violations[0], "期限 2026-08-05 が切れています")
			assert.Contains(t, invalid, "github.com/x/y@v1.2.3")
		})

		t.Run("上限を越えた期限は違反にし効力も失わせる", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"github.com/x/y@v1.2.3": {expires: today.AddDate(0, maxBypassMonths, 1), issue: 931, reason: "r", line: 3},
			}, current, today)
			require.Len(t, violations, 1)
			assert.Contains(t, violations[0], "を越えています")
			assert.Contains(t, invalid, "github.com/x/y@v1.2.3")
		})

		t.Run("go.mod に無いエントリは違反にする", func(t *testing.T) {
			t.Parallel()
			violations, _ := validateBypasses(map[string]bypass{
				"github.com/gone/away@v9.9.9": {expires: day("2026-09-06"), issue: 931, reason: "r", line: 3},
			}, current, today)
			require.Len(t, violations, 1)
			assert.Contains(t, violations[0], "go.mod に存在しません")
		})
	})
}

func Test_classify(t *testing.T) {
	t.Parallel()

	directFinding := finding{req: requirement{module: "d", version: "v1"}, ageDays: 1}
	indirectFinding := finding{req: requirement{module: "i", version: "v1", indirect: true}, ageDays: 1}
	findings := []finding{directFinding, indirectFinding}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("gate は direct だけを落とし indirect は報告に留める", func(t *testing.T) {
			t.Parallel()
			bypassed, blocked, reported := classify("gate", findings, nil, nil)
			assert.Empty(t, bypassed)
			require.Len(t, blocked, 1)
			assert.Equal(t, "d@v1", blocked[0].req.key())
			require.Len(t, reported, 1)
			assert.Equal(t, "i@v1", reported[0].req.key())
		})

		t.Run("audit は何も落とさない", func(t *testing.T) {
			t.Parallel()
			_, blocked, reported := classify("audit", findings, nil, nil)
			assert.Empty(t, blocked)
			assert.Len(t, reported, 2)
		})

		t.Run("有効なバイパスは gate を通す", func(t *testing.T) {
			t.Parallel()
			bypassed, blocked, _ := classify("gate", findings, map[string]bypass{
				"d@v1": {expires: day("2026-09-06"), issue: 931, reason: "r"},
			}, nil)
			require.Len(t, bypassed, 1)
			assert.Empty(t, blocked)
		})

		// 期限切れのバイパスが素通しするなら、期限そのものが何も担保しないことになる。
		t.Run("無効なバイパスは効力を失い gate が落とす", func(t *testing.T) {
			t.Parallel()
			bypassed, blocked, _ := classify("gate", findings, map[string]bypass{
				"d@v1": {expires: day("2026-01-01"), issue: 931, reason: "r"},
			}, map[string]struct{}{"d@v1": {}})
			assert.Empty(t, bypassed)
			require.Len(t, blocked, 1)
			assert.Equal(t, "d@v1", blocked[0].req.key())
		})
	})
}

func Test_summary(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("違反が無ければ対象件数だけを述べる", func(t *testing.T) {
			t.Parallel()
			got := summary("audit", nil, nil, nil, nil, nil, []requirement{{module: "a", version: "v1"}}, 7)
			assert.Contains(t, got, "対象 1 件 / 窓 7 日")
			assert.NotContains(t, got, "## cooldown 未達")
		})

		// 節が 1 つも無い出力を件数の行だけで終えると、検査が通ったのか節の組み立てが
		// 壊れて何も書けなかったのかを読者が区別できない。
		t.Run("節が 1 つも無ければ違反なしと明示する", func(t *testing.T) {
			t.Parallel()
			got := summary("audit", nil, nil, nil, nil, nil, nil, 7)
			assert.Equal(t, "対象 0 件 / 窓 7 日 / バイパス 0 件\n\n違反はありません。\n", got)
		})

		// gate が落とすものと落とさないものを同じ見出しに混ぜると、直す必要のある件が埋もれる。
		t.Run("ブロックしないものは参考の見出しへ分けて並べる", func(t *testing.T) {
			t.Parallel()
			f := finding{req: requirement{module: "i", version: "v1", indirect: true}, ageDays: 3}
			got := summary("gate", []finding{f}, nil, nil, nil, nil, []requirement{f.req}, 7)
			assert.Contains(t, got, "## 参考: 窓内だがブロックしないもの (1)")
			assert.Contains(t, got, "- i@v1 — 公開 3 日")
			assert.NotContains(t, got, "## cooldown 未達")
		})

		t.Run("ブロックした件を見出し付きで並べる", func(t *testing.T) {
			t.Parallel()
			f := finding{req: requirement{module: "d", version: "v1"}, published: day("2026-08-01"), ageDays: 5}
			got := summary("gate", []finding{f}, nil, nil, nil, nil, []requirement{f.req}, 7)
			assert.Contains(t, got, "## cooldown 未達 (1)")
			assert.Contains(t, got, "- d@v1 — 公開 5 日（2026-08-01）")
		})

		t.Run("バイパス設定の違反を見出し付きで並べる", func(t *testing.T) {
			t.Parallel()
			got := summary("audit", nil, nil, []string{"期限切れ"}, nil, nil, nil, 7)
			assert.Contains(t, got, "## バイパス設定の違反 (1)")
			assert.Contains(t, got, "期限切れ")
		})

		// 値は go.mod 由来で pull request が中身を決めるため、フェンスは値の側から長さを取る。
		t.Run("値に含まれるバッククォート連より長いフェンスで包む", func(t *testing.T) {
			t.Parallel()
			got := summary("audit", nil, nil, []string{"``` を含む理由"}, nil, nil, nil, 7)
			assert.Contains(t, got, "````text")
		})

		// 取得できなかったものを黙らせると、「見た結果 OK」と「見ていない」が区別できなくなる。
		t.Run("公開時刻を取得できなかったものを別枠で並べる", func(t *testing.T) {
			t.Parallel()
			got := summary("audit", nil, []requirement{{module: "p", version: "v1"}}, nil, nil, nil, nil, 7)
			assert.Contains(t, got, "## 公開時刻を取得できなかったもの (1)")
			assert.Contains(t, got, "- p@v1")
		})
	})
}

func Test_appendOutput(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("件数を key=value で追記する", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out")
			require.NoError(t, appendOutput(path, 2, 111, 1))

			body, err := os.ReadFile(path) //nolint:gosec // path は t.TempDir 配下
			require.NoError(t, err)
			assert.Equal(t, "findings=2\naudited=111\nblocking=1\n", string(body))
		})

		// 上書きすると同じジョブの先行ステップが書いた出力を消してしまう。
		t.Run("既にある内容へ追記する", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out")
			require.NoError(t, os.WriteFile(path, []byte("existing=1\n"), 0o600))
			require.NoError(t, appendOutput(path, 0, 3, 0))

			body, err := os.ReadFile(path) //nolint:gosec // path は t.TempDir 配下
			require.NoError(t, err)
			assert.Equal(t, "existing=1\nfindings=0\naudited=3\nblocking=0\n", string(body))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開けないパスはどの出力に失敗したかを含むエラーにする", func(t *testing.T) {
			t.Parallel()
			err := appendOutput(filepath.Join(t.TempDir(), "absent-dir", "out"), 1, 1, 1)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "open GITHUB_OUTPUT")
		})
	})
}

func Test_requirement_key(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("module と version を @ で繋ぐ", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "github.com/x/y@v1.2.3", requirement{module: "github.com/x/y", version: "v1.2.3"}.key())
		})

		t.Run("同じ module でも版が違えば別のキーになる", func(t *testing.T) {
			t.Parallel()
			a := requirement{module: "github.com/x/y", version: "v1.2.3"}
			b := requirement{module: "github.com/x/y", version: "v1.2.4"}
			assert.NotEqual(t, a.key(), b.key())
		})

		t.Run("indirect 注釈はキーに影響しない", func(t *testing.T) {
			t.Parallel()
			direct := requirement{module: "github.com/x/y", version: "v1.2.3"}
			indirect := requirement{module: "github.com/x/y", version: "v1.2.3", indirect: true}
			assert.Equal(t, direct.key(), indirect.key())
		})
	})
}

func Test_sortedKeys(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// バイパスの違反報告はこの順で並ぶため、map の反復順が漏れると報告順が実行ごとに変わる。
		t.Run("キーを辞書順で返す", func(t *testing.T) {
			t.Parallel()
			got := sortedKeys(map[string]bypass{"c@v1": {}, "a@v1": {}, "b@v1": {}})
			assert.Equal(t, []string{"a@v1", "b@v1", "c@v1"}, got)
		})

		t.Run("空のマップには空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, sortedKeys(map[string]bypass{}))
		})
	})
}

func Test_hasBypass(t *testing.T) {
	t.Parallel()

	req := requirement{module: "github.com/x/y", version: "v1.2.3"}
	bypasses := map[string]bypass{"github.com/x/y@v1.2.3": {expires: day("2026-09-06"), issue: 931, reason: "r"}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録があれば真を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, hasBypass(bypasses, map[string]struct{}{}, req))
		})

		t.Run("登録が無ければ偽を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, hasBypass(bypasses, map[string]struct{}{}, requirement{module: "other", version: "v1"}))
		})

		// 期限切れのバイパスが素通しするなら、期限そのものが何も担保しないことになる。
		t.Run("無効と判定されたキーは登録があっても偽を返す", func(t *testing.T) {
			t.Parallel()
			invalid := map[string]struct{}{"github.com/x/y@v1.2.3": {}}
			assert.False(t, hasBypass(bypasses, invalid, req))
		})
	})
}

func Test_fenceFor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("バッククォートを含まなければ CommonMark の下限 3 を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "```", fenceFor("普通の理由"))
		})

		// 値と同じ長さのフェンスを返すと、値の側がフェンスを閉じて Markdown を脱出できる。
		t.Run("3 連バッククォートを含む値には 4 連を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "````", fenceFor("``` を含む理由"))
		})

		t.Run("最長の連より 1 つ長い長さを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "``````", fenceFor("a `````"))
		})

		t.Run("連が途切れれば長さは数え直す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "```", fenceFor("`a`b`c`"))
		})
	})
}

func Test_fenced(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("見出しに件数を添えて本文をフェンスで包む", func(t *testing.T) {
			t.Parallel()
			var b strings.Builder
			fenced(&b, "cooldown 未達", []string{"- a@v1", "- b@v2"})
			assert.Equal(t, "## cooldown 未達 (2)\n\n```text\n- a@v1\n- b@v2\n```\n\n", b.String())
		})

		t.Run("本文が閉じられない長さのフェンスを使う", func(t *testing.T) {
			t.Parallel()
			var b strings.Builder
			fenced(&b, "見出し", []string{"``` を含む行"})
			assert.Contains(t, b.String(), "````text\n``` を含む行\n````")
		})
	})
}

func Test_publishedAt(t *testing.T) {
	t.Parallel()

	req := requirement{module: "github.com/BurntSushi/toml", version: "v1.4.0"}
	published := day("2026-08-01")
	client := func(fn roundTripFunc) *http.Client { return &http.Client{Transport: fn} }

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// 大文字を未エスケープのまま引くと proxy は 404 を返し、「存在しないバージョン」に化ける。
		t.Run("エスケープした .info を引き公開時刻を返す", func(t *testing.T) {
			t.Parallel()
			var requested string
			at, err := publishedAt(t.Context(), client(func(r *http.Request) (*http.Response, error) {
				requested = r.URL.String()
				return proxyResponse(http.StatusOK, infoBody(published)), nil
			}), req)
			require.NoError(t, err)
			assert.Equal(t, "https://proxy.golang.org/github.com/!burnt!sushi/toml/@v/v1.4.0.info", requested)
			assert.Equal(t, published, at.UTC())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("404 は proxy が知らないバージョンとして返す", func(t *testing.T) {
			t.Parallel()
			_, err := publishedAt(t.Context(), client(func(*http.Request) (*http.Response, error) {
				return proxyResponse(http.StatusNotFound, ""), nil
			}), req)
			require.ErrorIs(t, err, errProxyNotFound)
		})

		t.Run("410 も proxy が知らないバージョンとして返す", func(t *testing.T) {
			t.Parallel()
			_, err := publishedAt(t.Context(), client(func(*http.Request) (*http.Response, error) {
				return proxyResponse(http.StatusGone, ""), nil
			}), req)
			require.ErrorIs(t, err, errProxyNotFound)
		})

		// 500 を「知らないバージョン」と同じ扱いにすると、proxy 障害が検査の素通しになる。
		t.Run("想定外のステータスは取得失敗として区別する", func(t *testing.T) {
			t.Parallel()
			_, err := publishedAt(t.Context(), client(func(*http.Request) (*http.Response, error) {
				return proxyResponse(http.StatusInternalServerError, ""), nil
			}), req)
			require.ErrorIs(t, err, errProxyStatus)
			assert.Contains(t, err.Error(), "500")
		})

		t.Run("壊れた JSON はエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := publishedAt(t.Context(), client(func(*http.Request) (*http.Response, error) {
				return proxyResponse(http.StatusOK, "{"), nil
			}), req)
			require.Error(t, err)
		})

		t.Run("転送層のエラーを包んで返す", func(t *testing.T) {
			t.Parallel()
			_, err := publishedAt(t.Context(), client(func(*http.Request) (*http.Response, error) {
				return nil, errProxyStatus
			}), req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "fetch https://proxy.golang.org/")
		})

		// URL を組めない module path は「窓を満たした」でも「proxy が知らない」でもないため、
		// 取得失敗として上へ返す。
		t.Run("URL に使えない module path はリクエストを組めずエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := publishedAt(t.Context(), client(func(*http.Request) (*http.Response, error) {
				return proxyResponse(http.StatusOK, infoBody(published)), nil
			}), requirement{module: "github.com/x/\ny", version: "v1.0.0"})
			require.Error(t, err)
			assert.NotErrorIs(t, err, errProxyNotFound)
		})
	})
}

//nolint:paralleltest // プロセス共有の http.DefaultTransport を差し替えるため並列化不可
func Test_inspect(t *testing.T) {
	now := day("2026-08-06")
	targets := []requirement{
		{module: "old", version: "v1"},
		{module: "fresh", version: "v1"},
		{module: "edge", version: "v1"},
	}
	info := func(path string) string {
		switch {
		case strings.HasPrefix(path, "/fresh/"):
			return infoBody(now.AddDate(0, 0, -1))
		case strings.HasPrefix(path, "/edge/"):
			return infoBody(now.AddDate(0, 0, -7))
		default:
			return infoBody(now.AddDate(0, 0, -30))
		}
	}
	ages := func(path string) (int, string) { return http.StatusOK, info(path) }

	//nolint:paralleltest // 親が http.DefaultTransport を差し替えるため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が http.DefaultTransport を差し替えるため並列化不可
		t.Run("公開から窓日数ちょうどのものは含めず、窓に満たないものだけを返す", func(t *testing.T) {
			stubProxy(t, ages)
			findings, unresolved, err := inspect(t.Context(), targets, 7, now)
			require.NoError(t, err)
			assert.Empty(t, unresolved)
			require.Len(t, findings, 1)
			assert.Equal(t, "fresh@v1", findings[0].req.key())
			assert.Equal(t, 1, findings[0].ageDays)
		})

		//nolint:paralleltest // 親が http.DefaultTransport を差し替えるため並列化不可
		t.Run("経過日数の昇順に並べる", func(t *testing.T) {
			stubProxy(t, ages)
			findings, _, err := inspect(t.Context(), targets, 30, now)
			require.NoError(t, err)
			assert.Equal(t, []string{"fresh@v1", "edge@v1"}, keysOf([]requirement{findings[0].req, findings[1].req}))
		})

		// 打つ手の無い違反を作らないため、proxy に載らないものは失敗ではなく別枠へ回す。
		//nolint:paralleltest // 親が http.DefaultTransport を差し替えるため並列化不可
		t.Run("proxy が知らないバージョンは未解決として分ける", func(t *testing.T) {
			stubProxy(t, func(path string) (int, string) {
				if strings.HasPrefix(path, "/fresh/") {
					return http.StatusNotFound, ""
				}
				return ages(path)
			})
			findings, unresolved, err := inspect(t.Context(), targets, 7, now)
			require.NoError(t, err)
			assert.Empty(t, findings)
			assert.Equal(t, []string{"fresh@v1"}, keysOf(unresolved))
		})

		// 取得は並行に走るので、並べ替えを落とすと未解決の報告順が実行ごとに変わる。
		//nolint:paralleltest // 親が http.DefaultTransport を差し替えるため並列化不可
		t.Run("未解決はキーの辞書順に並べる", func(t *testing.T) {
			stubProxy(t, func(string) (int, string) { return http.StatusNotFound, "" })
			_, unresolved, err := inspect(t.Context(), targets, 7, now)
			require.NoError(t, err)
			assert.Equal(t, []string{"edge@v1", "fresh@v1", "old@v1"}, keysOf(unresolved))
		})
	})

	//nolint:paralleltest // 親が http.DefaultTransport を差し替えるため並列化不可
	t.Run("異常系", func(t *testing.T) {
		// 取得できないものを黙って通すと、「見た結果 OK」と「見ていない」を区別できなくなる。
		//nolint:paralleltest // 親が http.DefaultTransport を差し替えるため並列化不可
		t.Run("proxy の障害は検査全体の失敗にする", func(t *testing.T) {
			stubProxy(t, func(string) (int, string) { return http.StatusInternalServerError, "" })
			findings, unresolved, err := inspect(t.Context(), targets, 7, now)
			require.ErrorIs(t, err, errProxyStatus)
			assert.Empty(t, findings)
			assert.Empty(t, unresolved)
		})
	})
}

//nolint:paralleltest // プロセス共有の log 出力先を差し替えるため並列化不可
func Test_report(t *testing.T) {
	direct := finding{req: requirement{module: "d", version: "v1"}, ageDays: 1}
	indirect := finding{req: requirement{module: "i", version: "v1", indirect: true}, ageDays: 1}

	//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
	t.Run("正常系", func(t *testing.T) {
		// Go には解決時に窓を強制する機構が無く、gate が落とさなければ窓はどこにも存在しない。
		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("gate は direct の窓内違反を非ゼロ終了として数える", func(t *testing.T) {
			captureLog(t)
			got := report("gate", []finding{direct}, nil, nil, nil, nil, []requirement{direct.req}, 7, false)
			assert.Equal(t, 1, got)
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("gate でも indirect の窓内違反は数えない", func(t *testing.T) {
			captureLog(t)
			got := report("gate", []finding{indirect}, nil, nil, nil, nil, []requirement{indirect.req}, 7, false)
			assert.Equal(t, 0, got)
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("audit は窓内違反では落ちない", func(t *testing.T) {
			captureLog(t)
			got := report("audit", []finding{direct, indirect}, nil, nil, nil, nil, nil, 7, false)
			assert.Equal(t, 0, got)
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("バイパス設定の違反は audit でも数える", func(t *testing.T) {
			buf := captureLog(t)
			got := report("audit", nil, nil, []string{"期限が切れています"}, nil, nil, nil, 7, false)
			assert.Equal(t, 1, got)
			assert.Contains(t, buf.String(), "期限が切れています")
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("有効なバイパスは数えず通した事実を残す", func(t *testing.T) {
			buf := captureLog(t)
			bypasses := map[string]bypass{"d@v1": {expires: day("2026-09-06"), issue: 931, reason: "CRITICAL への即応"}}
			got := report("gate", []finding{direct}, nil, nil, bypasses, nil, []requirement{direct.req}, 7, false)
			assert.Equal(t, 0, got)
			assert.Contains(t, buf.String(), "#931 のバイパスで通します")
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("gate は公開時刻を取得できなかった direct だけを数える", func(t *testing.T) {
			captureLog(t)
			unresolved := []requirement{direct.req, indirect.req}
			got := report("gate", nil, unresolved, nil, nil, nil, unresolved, 7, false)
			assert.Equal(t, 1, got)
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("github 指定のときだけアノテーションを出す", func(t *testing.T) {
			buf := captureLog(t)
			report("gate", []finding{direct}, nil, nil, nil, nil, []requirement{direct.req}, 7, true)
			assert.Contains(t, buf.String(), "::error file=go.mod::d@v1 は公開 1 日で")
		})

		// 直すべき場所は go.mod ではなくバイパス lockfile なので、注釈の宛先を取り違えると
		// 差分行に印が付かず、レビューで気づけない。
		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("バイパス設定の違反はバイパス lockfile 宛のアノテーションにする", func(t *testing.T) {
			buf := captureLog(t)
			report("audit", nil, nil, []string{"期限が切れています"}, nil, nil, nil, 7, true)
			assert.Contains(t, buf.String(), "::error file="+bypassFile+"::期限が切れています")
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("報告に留めるものは error ではなく warning のアノテーションにする", func(t *testing.T) {
			buf := captureLog(t)
			report("gate", []finding{indirect}, nil, nil, nil, nil, []requirement{indirect.req}, 7, true)
			assert.Contains(t, buf.String(), "::warning file=go.mod::i@v1（indirect）は公開 1 日で")
			assert.NotContains(t, buf.String(), "::error")
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化不可
		t.Run("公開時刻を取得できなかった direct もアノテーションにする", func(t *testing.T) {
			buf := captureLog(t)
			unresolved := []requirement{direct.req}
			report("gate", nil, unresolved, nil, nil, nil, unresolved, 7, true)
			assert.Contains(t, buf.String(), "::error file=go.mod::d@v1 の公開時刻を module proxy から取得できませんでした")
		})
	})
}

//nolint:paralleltest // t.Chdir でプロセスの作業ディレクトリを変えるため並列化不可
func Test_added(t *testing.T) {
	current := []requirement{
		{module: "github.com/x/y", version: "v2.0.0"},
		{module: "github.com/kept/same", version: "v1.0.0"},
		{module: "github.com/new/one", version: "v0.1.0"},
	}

	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("base に無い組だけを対象にする", func(t *testing.T) {
			dir := t.TempDir()
			initGoModRepo(t, dir, "module m\n\nrequire (\n\tgithub.com/x/y v1.0.0\n\tgithub.com/kept/same v1.0.0\n)\n")
			t.Chdir(dir)

			got, err := added("HEAD", current)
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{"github.com/x/y@v2.0.0", "github.com/new/one@v0.1.0"}, keysOf(got))
		})
	})

	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("異常系", func(t *testing.T) {
		// base を取れないまま空の差分を返すと、検査が素通りしたことに誰も気づけない。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("base を解決できなければ取得手段を示してエラーにする", func(t *testing.T) {
			dir := t.TempDir()
			initGoModRepo(t, dir, "module m\n")
			t.Chdir(dir)

			_, err := added("no-such-ref", current)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "fetch-depth: 0")
		})

		// base を読めないまま差分を空として返すと、追加した依存がひとつも検査されない。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("base の go.mod を読み切れなければエラーにする", func(t *testing.T) {
			dir := t.TempDir()
			initGoModRepo(t, dir, "module m\n// "+strings.Repeat("a", bufio.MaxScanTokenSize)+"\n")
			t.Chdir(dir)

			got, err := added("HEAD", current)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "base の go.mod")
			assert.Empty(t, got)
		})
	})
}

func Test_parseArgs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグを省略すれば既定の窓を採る", func(t *testing.T) {
			t.Parallel()

			sub, opt, err := parseArgs([]string{"audit"})

			require.NoError(t, err)
			assert.Equal(t, "audit", sub)
			assert.Equal(t, options{window: defaultWindowDays}, opt)
		})

		t.Run("与えられたフラグを options へ載せる", func(t *testing.T) {
			t.Parallel()

			sub, opt, err := parseArgs([]string{"gate", "--base=origin/main", "--window-days=3", "--summary-out=s.md", "--github"})

			require.NoError(t, err)
			assert.Equal(t, "gate", sub)
			assert.Equal(t, options{base: "origin/main", summaryOut: "s.md", window: 3, github: true}, opt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブコマンドが無ければ使い方を示す", func(t *testing.T) {
			t.Parallel()

			_, _, err := parseArgs(nil)

			require.ErrorIs(t, err, errUsage)
			assert.Contains(t, err.Error(), "usage: go-cooldown <gate|audit>")
		})

		t.Run("未知のサブコマンドはエラーにする", func(t *testing.T) {
			t.Parallel()

			_, _, err := parseArgs([]string{"inspect"})

			require.ErrorIs(t, err, errUsage)
			assert.Contains(t, err.Error(), "未知のサブコマンド")
		})

		// base 無しの gate を通すと、比較対象が無いまま全 require を窓違反として落とす。
		t.Run("gate に base が無ければエラーにする", func(t *testing.T) {
			t.Parallel()

			_, _, err := parseArgs([]string{"gate"})

			require.ErrorIs(t, err, errUsage)
			assert.Contains(t, err.Error(), "--base")
		})

		// ヘルプ要求を解析失敗と同じ扱いにすると、`-h` が異常終了に化ける。
		t.Run("ヘルプ要求は flag.ErrHelp のまま返す", func(t *testing.T) {
			t.Parallel()

			_, _, err := parseArgs([]string{"audit", "-h"})

			require.ErrorIs(t, err, flag.ErrHelp)
		})

		t.Run("解釈できないフラグはエラーにする", func(t *testing.T) {
			t.Parallel()

			_, _, err := parseArgs([]string{"audit", "--no-such-flag"})

			require.Error(t, err)
			require.NotErrorIs(t, err, flag.ErrHelp)
			assert.Contains(t, err.Error(), "フラグを解釈できません")
		})
	})
}

//nolint:paralleltest // t.Chdir・captureLog・stubProxy がプロセス共有の状態を差し替えるため並列化不可
func Test_run(t *testing.T) {
	now := day("2026-08-06")
	fresh := day("2026-08-05")
	aged := day("2020-01-01")

	// 実行環境の GITHUB_OUTPUT を持ち込まないよう空にする。設定済みのランナー上で走らせると
	// 検査とは無関係なファイルへ追記してしまう。
	t.Setenv("GITHUB_OUTPUT", "")

	//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("audit は窓内の require を報告しつつ正常終了する", func(t *testing.T) {
			out := captureLog(t)
			useWorkTree(t, goModOneRequire)
			stubProxyPublishedAt(t, fresh)

			require.NoError(t, run([]string{"audit"}, now))
			assert.Contains(t, out.String(), "example.com/mod@v1.0.0（direct）は公開 1 日で")
		})

		// 窓の値を握り潰すと、gate が守る幅が既定値に固定されて設定が効かなくなる。
		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("window-days で窓を狭められる", func(t *testing.T) {
			out := captureLog(t)
			useWorkTree(t, goModOneRequire)
			stubProxyPublishedAt(t, fresh)

			require.NoError(t, run([]string{"audit", "--window-days=1"}, now))
			assert.NotContains(t, out.String(), "example.com/mod@v1.0.0")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("gate は base に無い require でも窓を越えていれば通す", func(t *testing.T) {
			out := captureLog(t)
			useGateRepo(t, "module m\n", goModOneRequire)
			stubProxyPublishedAt(t, aged)

			require.NoError(t, run([]string{"gate", "--base=HEAD"}, now))
			assert.Contains(t, out.String(), "対象 1 件")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("github 指定のときだけアノテーションを出す", func(t *testing.T) {
			out := captureLog(t)
			useWorkTree(t, goModOneRequire)
			stubProxyPublishedAt(t, fresh)

			require.NoError(t, run([]string{"audit", "--github"}, now))
			assert.Contains(t, out.String(), "::warning file=go.mod::example.com/mod@v1.0.0")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("summary-out 指定でサマリを書き出す", func(t *testing.T) {
			captureLog(t)
			dir := useWorkTree(t, goModOneRequire)
			stubProxyPublishedAt(t, fresh)
			path := filepath.Join(dir, "summary.md")

			require.NoError(t, run([]string{"audit", "--summary-out=" + path}, now))

			body, err := os.ReadFile(path) //nolint:gosec // テスト内の一時ファイル
			require.NoError(t, err)
			assert.Contains(t, string(body), "example.com/mod@v1.0.0")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("GITHUB_OUTPUT があれば件数を追記する", func(t *testing.T) {
			captureLog(t)
			dir := useWorkTree(t, goModOneRequire)
			stubProxyPublishedAt(t, fresh)
			path := filepath.Join(dir, "github-output")
			t.Setenv("GITHUB_OUTPUT", path)

			require.NoError(t, run([]string{"audit"}, now))

			body, err := os.ReadFile(path) //nolint:gosec // テスト内の一時ファイル
			require.NoError(t, err)
			assert.Contains(t, string(body), "findings=1")
			assert.Contains(t, string(body), "audited=1")
			assert.Contains(t, string(body), "blocking=0")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("ヘルプ要求は失敗にしない", func(t *testing.T) {
			captureLog(t)

			require.NoError(t, run([]string{"audit", "-h"}, now))
		})
	})

	//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
	t.Run("異常系", func(t *testing.T) {
		// Go には解決時に窓を強制する機構が無く、gate が落とさなければ窓はどこにも存在しない。
		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("gate は base に無い窓内の direct require を失敗にする", func(t *testing.T) {
			out := captureLog(t)
			useGateRepo(t, "module m\n", goModOneRequire)
			stubProxyPublishedAt(t, fresh)

			err := run([]string{"gate", "--base=HEAD"}, now)

			require.ErrorIs(t, err, errBlocking)
			assert.Contains(t, out.String(), "窓明けを待つか")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("期限切れのバイパスは audit でも失敗にする", func(t *testing.T) {
			captureLog(t)
			dir := useWorkTree(t, goModOneRequire)
			writeWorkTreeBypass(t, dir,
				`"example.com/mod@v1.0.0" = { expires = 2000-01-01, issue = 1, reason = "済んだ緊急対応" }`+"\n")
			stubProxyPublishedAt(t, aged)

			require.ErrorIs(t, run([]string{"audit"}, now), errBlocking)
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("使い方の誤りはそのまま失敗にする", func(t *testing.T) {
			captureLog(t)

			require.ErrorIs(t, run(nil, now), errUsage)
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("go.mod を読み切れなければ失敗にする", func(t *testing.T) {
			captureLog(t)
			useWorkTree(t, "module m\n// "+strings.Repeat("a", bufio.MaxScanTokenSize)+"\n")

			err := run([]string{"audit"}, now)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "go.mod")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("バイパス lockfile を解釈できなければ失敗にする", func(t *testing.T) {
			captureLog(t)
			dir := useWorkTree(t, goModOneRequire)
			writeWorkTreeBypass(t, dir, "これは行として解釈できない\n")

			require.ErrorIs(t, run([]string{"audit"}, now), errBypassInvalidLine)
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("gate の base を解決できなければ失敗にする", func(t *testing.T) {
			captureLog(t)
			useWorkTree(t, goModOneRequire)

			err := run([]string{"gate", "--base=no-such-ref"}, now)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "base との差分")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("公開時刻を取得できなければ失敗にする", func(t *testing.T) {
			captureLog(t)
			useWorkTree(t, goModOneRequire)
			stubProxy(t, func(string) (int, string) { return http.StatusInternalServerError, "" })

			err := run([]string{"audit"}, now)

			require.ErrorIs(t, err, errProxyStatus)
			assert.Contains(t, err.Error(), "公開時刻の取得")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("サマリを書き出せなければ失敗にする", func(t *testing.T) {
			captureLog(t)
			dir := useWorkTree(t, goModOneRequire)
			stubProxyPublishedAt(t, aged)

			err := run([]string{"audit", "--summary-out=" + dir}, now)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "write summary")
		})

		//nolint:paralleltest // 親がプロセス共有の状態を差し替えるため並列化不可
		t.Run("GITHUB_OUTPUT へ追記できなければ失敗にする", func(t *testing.T) {
			captureLog(t)
			dir := useWorkTree(t, goModOneRequire)
			stubProxyPublishedAt(t, aged)
			t.Setenv("GITHUB_OUTPUT", dir)

			err := run([]string{"audit"}, now)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "write GITHUB_OUTPUT")
		})
	})
}

//nolint:paralleltest // t.Chdir でプロセスの作業ディレクトリを変えるため並列化不可
func Test_readWorkTreeGoMod(t *testing.T) {
	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("正常系", func(t *testing.T) {
		// 検査対象は「今のツリーの go.mod」であって、base 側でも生成物でもない。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("作業ディレクトリの go.mod をそのまま返す", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModFixture), 0o600))
			t.Chdir(dir)

			got, err := readWorkTreeGoMod()

			require.NoError(t, err)
			assert.Equal(t, goModFixture, string(got))
		})
	})

	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("異常系", func(t *testing.T) {
		// 読めないことを黙って空の go.mod として扱うと、依存ゼロ＝違反ゼロとして素通りする。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("go.mod を読めなければ空を返さずエラーにする", func(t *testing.T) {
			t.Chdir(t.TempDir())

			got, err := readWorkTreeGoMod()

			require.ErrorIs(t, err, os.ErrNotExist)
			assert.Nil(t, got)
		})
	})
}
