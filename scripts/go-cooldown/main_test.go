package main

import (
	"os"
	"path/filepath"
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

func TestParseGoMod(t *testing.T) {
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
	})
}

func TestEscapePath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// 未エスケープの大文字は proxy が 404 を返し「存在しないバージョン」に化けるため、
		// この変換は取得可否そのものを決める。
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

func TestDiffAdded(t *testing.T) {
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

		// 版を上げたモジュールは別のキーとして現れるので、追加と更新を 1 つの判定で拾える。
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

func TestReadBypasses(t *testing.T) {
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

		// 読み飛ばしを許すと、書き損じたエントリが「存在しない」状態を警告なく作る。
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

		// 後勝ちの上書きを許すと、どちらのエントリが効くかが行順で決まる。
		t.Run("キーの重複はエラーにする", func(t *testing.T) {
			t.Parallel()
			line := `"github.com/x/y@v1.2.3" = { expires = 2026-11-06, issue = 931, reason = "r" }` + "\n"
			_, err := readBypasses(writeBypass(t, line+line))
			require.Error(t, err)
			assert.ErrorIs(t, err, errBypassDuplicateKey)
		})
	})
}

func TestValidateBypasses(t *testing.T) {
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

		// 放置されたバイパスは恒久 allowlist と区別が付かなくなるので、期限で必ず回収する。
		t.Run("期限切れは違反にし効力も失わせる", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"github.com/x/y@v1.2.3": {expires: day("2026-08-05"), issue: 931, reason: "r", line: 3},
			}, current, today)
			require.Len(t, violations, 1)
			assert.Contains(t, violations[0], "期限 2026-08-05 が切れています")
			assert.Contains(t, invalid, "github.com/x/y@v1.2.3")
		})

		// 期限を遠い未来へ置くだけで、期限切れ検査を素通りする恒久 allowlist を作れてしまう。
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

func TestClassify(t *testing.T) {
	t.Parallel()

	directFinding := finding{req: requirement{module: "d", version: "v1"}, ageDays: 1}
	indirectFinding := finding{req: requirement{module: "i", version: "v1", indirect: true}, ageDays: 1}
	findings := []finding{directFinding, indirectFinding}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// indirect の版は MVS で direct の要求下限に縛られ、落としても打つ手が無いことがある。
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

func TestSummary(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("違反が無ければ対象件数だけを述べる", func(t *testing.T) {
			t.Parallel()
			got := summary("audit", nil, nil, nil, nil, nil, []requirement{{module: "a", version: "v1"}}, 7)
			assert.Contains(t, got, "対象 1 件 / 窓 7 日")
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

func TestAppendOutput(t *testing.T) {
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
	})
}
