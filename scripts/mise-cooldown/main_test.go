package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// miseFixture は実際の mise.toml の形を最小で再現する。`[settings]` と `[env]` を混ぜてあるのは、
// ツール宣言だけを読めていることを確かめるため。
const miseFixture = `min_version = "2026.6.0"

[settings]
pipx.uvx = true

[tools]
go = "1.26.5"
sqlc = "1.31.1"
"aqua:golang-migrate/migrate" = "4.19.1"
"go:go.uber.org/mock/mockgen" = "0.6.0"
"npm:@redocly/cli" = "2.31.4"
"pipx:graphifyy[sql]" = "0.9.25"

[env]
OTEL_LGTM_VERSION = "0.28.0"
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

func TestParseTools(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("裸のキーと引用符付きのキーの両方を読む", func(t *testing.T) {
			t.Parallel()
			got, err := parseTools([]byte(miseFixture))
			require.NoError(t, err)

			ids := make([]string, 0, len(got))
			for _, tool := range got {
				ids = append(ids, tool.id())
			}
			assert.ElementsMatch(t, []string{
				"go@1.26.5",
				"sqlc@1.31.1",
				"aqua:golang-migrate/migrate@4.19.1",
				"go:go.uber.org/mock/mockgen@0.6.0",
				"npm:@redocly/cli@2.31.4",
				"pipx:graphifyy[sql]@0.9.25",
			}, ids)
		})

		// `[settings]` の pipx.uvx や `[env]` のバージョン値をツールとして拾うと、解決できない
		// キーが毎回 unresolved として報告され続ける。
		t.Run("tools 以外のセクションからは読まない", func(t *testing.T) {
			t.Parallel()
			got, err := parseTools([]byte(miseFixture))
			require.NoError(t, err)
			for _, tool := range got {
				assert.NotEqual(t, "OTEL_LGTM_VERSION", tool.key)
				assert.NotEqual(t, "min_version", tool.key)
			}
		})
	})
}

func TestDiffAdded(t *testing.T) {
	t.Parallel()

	before := []tool{{key: "a", version: "1.0.0"}, {key: "b", version: "1.0.0"}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("新規追加のみを返す", func(t *testing.T) {
			t.Parallel()
			got := diffAdded(before, append(before, tool{key: "c", version: "0.1.0"}))
			require.Len(t, got, 1)
			assert.Equal(t, "c@0.1.0", got[0].id())
		})

		t.Run("バージョンを上げたツールも対象にする", func(t *testing.T) {
			t.Parallel()
			got := diffAdded(before, []tool{{key: "a", version: "1.0.0"}, {key: "b", version: "2.0.0"}})
			require.Len(t, got, 1)
			assert.Equal(t, "b@2.0.0", got[0].id())
		})

		t.Run("変化が無ければ空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, diffAdded(before, before))
		})
	})
}

func TestBackendKindAndWindow(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// GitHub リリースは tag の付け替えが起こり得るぶん、pin-actions / pin-images と同じ窓を採る。
		t.Run("GitHub リリース系の窓は 14 日", func(t *testing.T) {
			t.Parallel()
			for _, backend := range []string{"aqua:owner/repo", "ubi:owner/repo", "github:owner/repo"} {
				assert.Equal(t, "github", backendKind(backend), backend)
				assert.Equal(t, releaseWindowDays, windowFor(backend), backend)
			}
		})

		t.Run("パッケージレジストリ系の窓は 7 日", func(t *testing.T) {
			t.Parallel()
			for backend, kind := range map[string]string{
				"go:go.uber.org/mock": "go",
				"npm:markdownlint":    "npm",
				"pipx:sqlfluff":       "pypi",
				"uvx:sqlfluff":        "pypi",
			} {
				assert.Equal(t, kind, backendKind(backend), backend)
				assert.Equal(t, registryWindowDays, windowFor(backend), backend)
			}
		})

		t.Run("取得経路を持たない backend は空種別になる", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, backendKind("asdf:someone/plugin"))
		})
	})
}

func TestEscapeModulePath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// 未エスケープの大文字は proxy が 404 を返し「存在しないバージョン」に化ける。
		t.Run("大文字を ! + 小文字へ変換する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "github.com/!burnt!sushi/toml", escapeModulePath("github.com/BurntSushi/toml"))
		})

		t.Run("小文字だけのパスは変えない", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "go.uber.org/mock", escapeModulePath("go.uber.org/mock"))
		})
	})
}

func TestExtrasStripping(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// PyPI は extras を知らないので、付けたまま問い合わせると存在しないパッケージになる。
		t.Run("pipx の extras を落とす", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "graphifyy", extrasRe.ReplaceAllString("graphifyy[sql]", ""))
		})

		t.Run("extras が無ければ変えない", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "sqlfluff", extrasRe.ReplaceAllString("sqlfluff", ""))
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
				`"aqua:owner/repo@1.2.3" = { expires = 2026-11-06, issue = 931, reason = "CRITICAL への即応" }`+"\n")
			got, err := readBypasses(path)
			require.NoError(t, err)
			require.Len(t, got, 1)

			b := got["aqua:owner/repo@1.2.3"]
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
			_, err := readBypasses(writeBypass(t, `"a@1" = { issue = 1, reason = "期限が無い" }`+"\n"))
			require.Error(t, err)
			assert.ErrorIs(t, err, errBypassInvalidLine)
		})

		// 後勝ちの上書きを許すと、どちらのエントリが効くかが行順で決まる。
		t.Run("キーの重複はエラーにする", func(t *testing.T) {
			t.Parallel()
			line := `"a@1" = { expires = 2026-11-06, issue = 1, reason = "r" }` + "\n"
			_, err := readBypasses(writeBypass(t, line+line))
			require.Error(t, err)
			assert.ErrorIs(t, err, errBypassDuplicateKey)
		})
	})
}

func TestValidateBypasses(t *testing.T) {
	t.Parallel()

	today := day("2026-08-06")
	declared := []tool{{key: "aqua:owner/repo", version: "1.2.3"}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限内かつ mise.toml にあるエントリは違反にならない", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"aqua:owner/repo@1.2.3": {expires: day("2026-09-06"), issue: 1, reason: "r", line: 1},
			}, declared, today)
			assert.Empty(t, violations)
			assert.Empty(t, invalid)
		})

		t.Run("期限当日と上限ちょうどは許す", func(t *testing.T) {
			t.Parallel()
			for _, expires := range []time.Time{today, today.AddDate(0, maxBypassMonths, 0)} {
				violations, invalid := validateBypasses(map[string]bypass{
					"aqua:owner/repo@1.2.3": {expires: expires, issue: 1, reason: "r", line: 1},
				}, declared, today)
				assert.Empty(t, violations)
				assert.Empty(t, invalid)
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// 放置されたバイパスは恒久 allowlist と区別が付かなくなるので、期限で必ず回収する。
		t.Run("期限切れは違反にし効力も失わせる", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"aqua:owner/repo@1.2.3": {expires: day("2026-08-05"), issue: 1, reason: "r", line: 3},
			}, declared, today)
			require.Len(t, violations, 1)
			assert.Contains(t, violations[0], "が切れています")
			assert.Contains(t, invalid, "aqua:owner/repo@1.2.3")
		})

		// 期限を遠い未来へ置くだけで、期限切れ検査を素通りする恒久 allowlist を作れてしまう。
		t.Run("上限を越えた期限は違反にし効力も失わせる", func(t *testing.T) {
			t.Parallel()
			violations, invalid := validateBypasses(map[string]bypass{
				"aqua:owner/repo@1.2.3": {expires: today.AddDate(0, maxBypassMonths, 1), issue: 1, reason: "r", line: 3},
			}, declared, today)
			require.Len(t, violations, 1)
			assert.Contains(t, violations[0], "を越えています")
			assert.Contains(t, invalid, "aqua:owner/repo@1.2.3")
		})

		t.Run("mise.toml に無いエントリは違反にする", func(t *testing.T) {
			t.Parallel()
			violations, _ := validateBypasses(map[string]bypass{
				"aqua:gone/away@9.9.9": {expires: day("2026-09-06"), issue: 1, reason: "r", line: 3},
			}, declared, today)
			require.Len(t, violations, 1)
			assert.Contains(t, violations[0], "に存在しません")
		})
	})
}

func TestClassify(t *testing.T) {
	t.Parallel()

	f := finding{tool: tool{key: "aqua:owner/repo", version: "1.2.3", backend: "aqua:owner/repo"}, ageDays: 1, window: 14}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("gate は窓内をすべて落とす", func(t *testing.T) {
			t.Parallel()
			_, blocked, reported := classify("gate", []finding{f}, nil, nil)
			require.Len(t, blocked, 1)
			assert.Empty(t, reported)
		})

		t.Run("audit は何も落とさない", func(t *testing.T) {
			t.Parallel()
			_, blocked, reported := classify("audit", []finding{f}, nil, nil)
			assert.Empty(t, blocked)
			assert.Len(t, reported, 1)
		})

		t.Run("有効なバイパスは gate を通す", func(t *testing.T) {
			t.Parallel()
			bypassed, blocked, _ := classify("gate", []finding{f}, map[string]bypass{
				"aqua:owner/repo@1.2.3": {expires: day("2026-09-06"), issue: 1, reason: "r"},
			}, nil)
			require.Len(t, bypassed, 1)
			assert.Empty(t, blocked)
		})

		// 期限切れのバイパスが素通しするなら、期限そのものが何も担保しないことになる。
		t.Run("無効なバイパスは効力を失い gate が落とす", func(t *testing.T) {
			t.Parallel()
			bypassed, blocked, _ := classify("gate", []finding{f}, map[string]bypass{
				"aqua:owner/repo@1.2.3": {expires: day("2026-01-01"), issue: 1, reason: "r"},
			}, map[string]struct{}{"aqua:owner/repo@1.2.3": {}})
			assert.Empty(t, bypassed)
			require.Len(t, blocked, 1)
		})
	})
}

func TestSummary(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("除外件数と窓を先頭に述べる", func(t *testing.T) {
			t.Parallel()
			got := summary("audit", nil, nil, []tool{{key: "go", version: "1.26.5"}}, nil, nil, nil, nil)
			assert.Contains(t, got, "ランタイム除外 1 件")
			assert.Contains(t, got, "窓 GitHub 14 日・レジストリ 7 日")
		})

		t.Run("ブロックした件を見出し付きで並べる", func(t *testing.T) {
			t.Parallel()
			f := finding{
				tool:      tool{key: "aqua:owner/repo", version: "1.2.3", backend: "aqua:owner/repo"},
				published: day("2026-08-01"), ageDays: 5, window: 14,
			}
			got := summary("gate", []finding{f}, nil, nil, nil, nil, nil, []tool{f.tool})
			assert.Contains(t, got, "## cooldown 未達 (1)")
			assert.Contains(t, got, "aqua:owner/repo@1.2.3")
		})

		// 値は mise.toml 由来で pull request が中身を決めるため、フェンスは値の側から長さを取る。
		t.Run("値に含まれるバッククォート連より長いフェンスで包む", func(t *testing.T) {
			t.Parallel()
			got := summary("audit", nil, nil, nil, []string{"``` を含む理由"}, nil, nil, nil)
			assert.Contains(t, got, "````text")
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
			require.NoError(t, appendOutput(path, 2, 33, 1))

			body, err := os.ReadFile(path) //nolint:gosec // path は t.TempDir 配下
			require.NoError(t, err)
			assert.Equal(t, "findings=2\naudited=33\nblocking=1\n", string(body))
		})
	})
}
