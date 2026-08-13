package main

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// fullMiseTOML は、testVersions() と同じ内容を持つ伝播元。
	fullMiseTOML = "min_version = \"2099.1.1\"\n" +
		"\n[env]\nOTEL_LGTM_VERSION = \"0.99.0\"\n" +
		"\n[tools]\ngo = \"1.99.0\"\nnode = \"22.99.0\"\npython = \"3.99.0\"\n"
	// serverDockerfile は、golang の FROM を期待件数だけ持つサーバ用 Dockerfile。
	serverDockerfile = "FROM golang:0.0.0-alpine AS base\n" +
		"FROM golang:0.0.0-alpine AS builder\n" +
		"FROM golang:0.0.0-alpine AS runtime\n" +
		"ARG MISE_VERSION=v0.0.0\n"
	// toolsDockerfile は、golang / node / python の FROM を期待件数だけ持つツール用 Dockerfile。
	toolsDockerfile = "FROM golang:0.0.0-alpine AS go-tools\n" +
		"FROM golang:0.0.0-alpine AS go-runtime\n" +
		"FROM node:0.0.0-alpine AS node-tools\n" +
		"FROM python:0.0.0-slim AS python-tools\n" +
		"ARG MISE_VERSION=v0.0.0\n"
	// composeYAML は、otel-lgtm のイメージタグを 1 件持つ compose 定義。
	composeYAML = "services:\n  otel_lgtm:\n    image: grafana/otel-lgtm:0.0.0\n"
	// imageReadme は、件数の下限が無い README。存在だけを満たします。
	imageReadme = "`golang:0.0.0-alpine` / `node:0.0.0-alpine` / `python:0.0.0-slim`\n"
)

// errFakeGetwd は、差し替えた作業ディレクトリの取得手段が返す失敗。
var errFakeGetwd = xerrors.New("fake getwd failed")

// testVersions は、どのフィールドが誤って伝播しても判別できるよう値を散らしたバージョン集合です。
func testVersions() runtimeVersions {
	return runtimeVersions{
		Go:       "1.99.0",
		Node:     "22.99.0",
		Python:   "3.99.0",
		Mise:     "2099.1.1",
		OtelLgtm: "0.99.0",
	}
}

// writeFile は root からの相対パスへ body を書き出し、その絶対パスを返します。
func writeFile(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

// versionByLabel は rule 一覧を label→version の対応表へ畳み込みます。
func versionByLabel(rules []rule) map[string]string {
	got := make(map[string]string, len(rules))
	for _, r := range rules {
		got[r.label] = r.version
	}

	return got
}

// ruleByLabel は label に一致する rule を返します。
func ruleByLabel(t *testing.T, rules []rule, label string) rule {
	t.Helper()

	for _, r := range rules {
		if r.label == label {
			return r
		}
	}

	t.Fatalf("rule not found: %s", label)

	return rule{}
}

func Test_parseMiseTOML(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ルート・env・tools の各セクションから伝播元の版を抽出する", func(t *testing.T) {
			t.Parallel()
			body := "min_version = \"2099.1.1\"\n" +
				"\n" +
				"[env]\n" +
				"OTEL_LGTM_VERSION = \"0.99.0\"\n" +
				"\n" +
				"[tools]\n" +
				"go = \"1.99.0\"\n" +
				"node = \"22.99.0\"\n" +
				"python = \"3.99.0\"\n"

			got, err := parseMiseTOML(writeFile(t, t.TempDir(), "mise.toml", body))

			require.NoError(t, err)
			assert.Equal(t, testVersions(), got)
		})

		t.Run("コメント行と空行は読み飛ばす", func(t *testing.T) {
			t.Parallel()
			body := "# tools は mise が管理する\n" +
				"\n" +
				"[tools]\n" +
				"# go = \"0.0.0\"\n" +
				"go = \"1.99.0\"\n"

			got, err := parseMiseTOML(writeFile(t, t.TempDir(), "mise.toml", body))

			require.NoError(t, err)
			assert.Equal(t, "1.99.0", got.Go)
		})

		t.Run("対象外セクションの同名キーは取り込まない", func(t *testing.T) {
			t.Parallel()
			body := "[tools]\n" +
				"go = \"1.99.0\"\n" +
				"\n" +
				"[tasks]\n" +
				"go = \"0.0.0\"\n"

			got, err := parseMiseTOML(writeFile(t, t.TempDir(), "mise.toml", body))

			require.NoError(t, err)
			assert.Equal(t, "1.99.0", got.Go)
		})

		t.Run("対象キーが 1 つも無ければ全て空のまま返す", func(t *testing.T) {
			t.Parallel()

			got, err := parseMiseTOML(writeFile(t, t.TempDir(), "mise.toml", "[tools]\nrust = \"1.0.0\"\n"))

			require.NoError(t, err)
			assert.Equal(t, runtimeVersions{}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("mise.toml が読めなければエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := parseMiseTOML(filepath.Join(t.TempDir(), "absent.toml"))

			require.ErrorContains(t, err, "open")
		})

		t.Run("1 行が走査上限を超える mise.toml は読めたところまでで成功にせずエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "[tools]\ngo = \"1.99.0\"\n" + strings.Repeat("x", bufio.MaxScanTokenSize+1) + "\n"

			_, err := parseMiseTOML(writeFile(t, t.TempDir(), "mise.toml", body))

			require.ErrorIs(t, err, bufio.ErrTooLong)
			require.ErrorContains(t, err, "scan")
		})
	})
}

func Test_applyMiseKV(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("tools の go / node / python をそれぞれのフィールドへ入れる", func(t *testing.T) {
			t.Parallel()
			var v runtimeVersions

			applyMiseKV(&v, "tools", "go", "1.99.0")
			applyMiseKV(&v, "tools", "node", "22.99.0")
			applyMiseKV(&v, "tools", "python", "3.99.0")

			assert.Equal(t, runtimeVersions{Go: "1.99.0", Node: "22.99.0", Python: "3.99.0"}, v)
		})

		t.Run("ルートの min_version は mise 自身の版として入れる", func(t *testing.T) {
			t.Parallel()
			var v runtimeVersions

			applyMiseKV(&v, "", "min_version", "2099.1.1")

			assert.Equal(t, "2099.1.1", v.Mise)
		})

		t.Run("env の OTEL_LGTM_VERSION は otel-lgtm の版として入れる", func(t *testing.T) {
			t.Parallel()
			var v runtimeVersions

			applyMiseKV(&v, "env", "OTEL_LGTM_VERSION", "0.99.0")

			assert.Equal(t, "0.99.0", v.OtelLgtm)
		})

		t.Run("同じキーでもセクションが違えば取り込まない", func(t *testing.T) {
			t.Parallel()
			var v runtimeVersions

			applyMiseKV(&v, "env", "go", "0.0.0")
			applyMiseKV(&v, "tools", "min_version", "0.0.0")
			applyMiseKV(&v, "", "OTEL_LGTM_VERSION", "0.0.0")

			assert.Equal(t, runtimeVersions{}, v)
		})

		t.Run("対象外のキーは無視する", func(t *testing.T) {
			t.Parallel()
			var v runtimeVersions

			applyMiseKV(&v, "tools", "rust", "1.0.0")

			assert.Equal(t, runtimeVersions{}, v)
		})
	})
}

//nolint:paralleltest // log の出力先を差し替えるため並列化できない
func Test_printSource(t *testing.T) {
	//nolint:paralleltest // 親が log の出力先を差し替えるため並列化できない
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化できない
		t.Run("伝播元として読み取った版を全て出力する", func(t *testing.T) {
			out := captureLog(t, func() { printSource(testVersions()) })

			assert.Contains(t, out, "Source: mise.toml")
			assert.Contains(t, out, "go          = 1.99.0")
			assert.Contains(t, out, "node        = 22.99.0")
			assert.Contains(t, out, "python      = 3.99.0")
			assert.Contains(t, out, "min_version = 2099.1.1")
			assert.Contains(t, out, "otel-lgtm   = 0.99.0")
		})

		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化できない
		t.Run("読み取れなかった版は空欄にせず未設定と分かる形で出力する", func(t *testing.T) {
			out := captureLog(t, func() { printSource(runtimeVersions{Go: "1.99.0"}) })

			assert.Contains(t, out, "node        = (unset)")
			assert.Contains(t, out, "otel-lgtm   = (unset)")
		})
	})
}

// captureLog は fn の実行中に log へ出力された内容を集めます。
func captureLog(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	fn()

	return buf.String()
}

func Test_dockerfileRule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("記載件数の下限を保持して drift を検出できるようにする", func(t *testing.T) {
			t.Parallel()

			r := dockerfileRule("docker/tools/Dockerfile", "tools (golang base)", golangFromRe, "1.99.0", 2)

			assert.Equal(t, 2, r.expectedCount)
			assert.Equal(t, "docker/tools/Dockerfile", r.file)
			assert.Equal(t, "tools (golang base)", r.label)
			assert.Equal(t, "1.99.0", r.version)
		})

		t.Run("version 部分だけを置き換える replace を持つ", func(t *testing.T) {
			t.Parallel()

			r := dockerfileRule("docker/tools/Dockerfile", "tools (golang base)", golangFromRe, "1.99.0", 2)

			assert.Equal(t, "FROM golang:1.99.0-alpine", r.replace("FROM golang:1.26.5-alpine"))
		})
	})
}

func Test_readmeRule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("記載件数の下限を設けず書き方の変化で abort させない", func(t *testing.T) {
			t.Parallel()

			r := readmeRule("docker/README.md", "docker/README.md (golang image)", golangImageRe, "1.99.0")

			assert.Equal(t, 0, r.expectedCount)
			assert.Equal(t, "docker/README.md", r.file)
			assert.Equal(t, "1.99.0", r.version)
		})

		t.Run("バッククォートを保ったまま version 部分だけを置き換える", func(t *testing.T) {
			t.Parallel()

			r := readmeRule("docker/README.md", "docker/README.md (golang image)", golangImageRe, "1.99.0")

			assert.Equal(t, "`golang:1.99.0-alpine`", r.replace("`golang:1.26.5-alpine`"))
		})
	})
}

func Test_buildRules(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("各書き換え先へ対応するランタイムの版を割り当てる", func(t *testing.T) {
			t.Parallel()
			v := testVersions()
			want := map[string]string{
				"go.mod (go directive)":                     v.Go,
				"docker/server/Dockerfile (golang base)":    v.Go,
				"docker/tools/Dockerfile (golang base)":     v.Go,
				"docker/tools/Dockerfile (node base)":       v.Node,
				"docker/tools/Dockerfile (python base)":     v.Python,
				"docker/README.md (golang image)":           v.Go,
				"docker/README.md (node image)":             v.Node,
				"docker/README.md (python image)":           v.Python,
				"docker/README.ja.md (golang image)":        v.Go,
				"docker/README.ja.md (node image)":          v.Node,
				"docker/README.ja.md (python image)":        v.Python,
				"docker/server/README.md (golang image)":    v.Go,
				"docker/server/README.ja.md (golang image)": v.Go,
				"docker/tools/README.md (golang image)":     v.Go,
				"docker/tools/README.md (node image)":       v.Node,
				"docker/tools/README.md (python image)":     v.Python,
				"docker/tools/README.ja.md (golang image)":  v.Go,
				"docker/tools/README.ja.md (node image)":    v.Node,
				"docker/tools/README.ja.md (python image)":  v.Python,
				"docker/tools/Dockerfile (mise version)":    v.Mise,
				"docker/server/Dockerfile (mise version)":   v.Mise,
				"docker-compose.yaml (otel-lgtm image)":     v.OtelLgtm,
			}

			rules := buildRules(v)

			assert.Equal(t, want, versionByLabel(rules))
			assert.Len(t, rules, len(want), "label が重複すると対応表から落ちるため件数も確かめる")
		})

		t.Run("go.mod の rule は go ディレクティブを丸ごと書き換える", func(t *testing.T) {
			t.Parallel()

			r := ruleByLabel(t, buildRules(testVersions()), "go.mod (go directive)")

			assert.Equal(t, 1, r.expectedCount)
			assert.Equal(t, "go 1.99.0", r.replace("go 1.26.5"))
		})

		t.Run("Dockerfile の golang 記載は現行の出現箇所数を下限にする", func(t *testing.T) {
			t.Parallel()
			rules := buildRules(testVersions())

			server := ruleByLabel(t, rules, "docker/server/Dockerfile (golang base)")
			tools := ruleByLabel(t, rules, "docker/tools/Dockerfile (golang base)")

			assert.Equal(t, serverDockerfileGolangCount, server.expectedCount)
			assert.Equal(t, toolsDockerfileGolangCount, tools.expectedCount)
		})
	})
}

func Test_fromReplacer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("version の前後を保ったまま置き換える", func(t *testing.T) {
			t.Parallel()

			replace := fromReplacer(nodeFromRe, "22.99.0")

			assert.Equal(t, "FROM node:22.99.0-bookworm-slim", replace("FROM node:20.1.0-bookworm-slim"))
		})

		t.Run("末尾の capture が空でも version だけを置き換える", func(t *testing.T) {
			t.Parallel()

			replace := fromReplacer(miseDockerfileRe, "2099.1.1")

			assert.Equal(t, "MISE_VERSION=v2099.1.1", replace("MISE_VERSION=v2025.1.1"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("capture group が足りない regex では書き換えずに元の文字列を返す", func(t *testing.T) {
			t.Parallel()

			replace := fromReplacer(regexp.MustCompile(`golang:\d+\.\d+\.\d+`), "1.99.0")

			assert.Equal(t, "golang:1.26.5", replace("golang:1.26.5"))
		})
	})
}

func Test_validateRules(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("版が揃い対象ファイルも存在すれば報告しない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "docker/tools/Dockerfile", "FROM golang:1.26.5-alpine\n")
			rules := []rule{dockerfileRule("docker/tools/Dockerfile", "tools", golangFromRe, "1.99.0", 1)}

			assert.Empty(t, validateRules(rules, root))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("mise.toml に対応する版が無い rule を報告する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "docker/tools/Dockerfile", "FROM golang:1.26.5-alpine\n")
			rules := []rule{dockerfileRule("docker/tools/Dockerfile", "tools", golangFromRe, "", 1)}

			errs := validateRules(rules, root)

			require.Len(t, errs, 1)
			assert.Contains(t, errs[0], "version が未設定")
		})

		t.Run("対象ファイルが存在しない rule を報告する", func(t *testing.T) {
			t.Parallel()
			rules := []rule{dockerfileRule("docker/tools/Dockerfile", "tools", golangFromRe, "1.99.0", 1)}

			errs := validateRules(rules, t.TempDir())

			require.Len(t, errs, 1)
			assert.Contains(t, errs[0], "docker/tools/Dockerfile が見つかりません")
		})

		t.Run("版の未設定とファイル不在は両方まとめて報告する", func(t *testing.T) {
			t.Parallel()
			rules := []rule{dockerfileRule("docker/tools/Dockerfile", "tools", golangFromRe, "", 1)}

			errs := validateRules(rules, t.TempDir())

			assert.Len(t, errs, 2)
		})
	})
}

func Test_computeChanges(t *testing.T) {
	t.Parallel()

	const dockerfile = "# builder\n" +
		"FROM golang:1.26.5-alpine AS builder\n" +
		"# runtime\n" +
		"FROM golang:1.26.5-alpine AS runtime\n" +
		"# mise\n" +
		"ENV MISE_VERSION=v2025.1.1\n"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一ファイルへの複数 rule を積み上げて 1 つの内容にする", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "Dockerfile", dockerfile)
			rules := []rule{
				dockerfileRule("Dockerfile", "golang base", golangFromRe, "1.99.0", 2),
				dockerfileRule("Dockerfile", "mise version", miseDockerfileRe, "2099.1.1", 1),
			}

			states, errs := computeChanges(rules, root)

			require.Empty(t, errs)
			require.Contains(t, states, "Dockerfile")
			assert.Equal(t, "# builder\n"+
				"FROM golang:1.99.0-alpine AS builder\n"+
				"# runtime\n"+
				"FROM golang:1.99.0-alpine AS runtime\n"+
				"# mise\n"+
				"ENV MISE_VERSION=v2099.1.1\n", states["Dockerfile"].current)
		})

		t.Run("FROM の間にコメントが無くても行を巻き込まずに置換する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "Dockerfile", "ARG MISE_VERSION\n"+
				"FROM golang:1.26.5-alpine AS builder\n"+
				"RUN go build ./...\n"+
				"FROM golang:1.26.5-alpine AS runtime\n")
			rules := []rule{dockerfileRule("Dockerfile", "golang base", golangFromRe, "1.99.0", 2)}

			states, errs := computeChanges(rules, root)

			require.Empty(t, errs)
			assert.Equal(t, "ARG MISE_VERSION\n"+
				"FROM golang:1.99.0-alpine AS builder\n"+
				"RUN go build ./...\n"+
				"FROM golang:1.99.0-alpine AS runtime\n", states["Dockerfile"].current)
		})

		t.Run("適用した rule とマッチ件数を記録する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "Dockerfile", dockerfile)
			rules := []rule{dockerfileRule("Dockerfile", "golang base", golangFromRe, "1.99.0", 2)}

			states, errs := computeChanges(rules, root)

			require.Empty(t, errs)
			assert.Equal(t, []string{"golang base x2"}, states["Dockerfile"].applied)
			assert.Equal(t, dockerfile, states["Dockerfile"].original, "元の内容は書き換え前として保持する")
		})

		t.Run("下限を設けない rule は記載が 1 つも無くても通す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "README.md", "golang のイメージには触れていない\n")
			rules := []rule{readmeRule("README.md", "README.md (golang image)", golangImageRe, "1.99.0")}

			states, errs := computeChanges(rules, root)

			require.Empty(t, errs)
			assert.Equal(t, states["README.md"].original, states["README.md"].current)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("マッチ件数が下限未満なら書き換えずに報告する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "Dockerfile", dockerfile)
			rules := []rule{dockerfileRule("Dockerfile", "golang base", golangFromRe, "1.99.0", 3)}

			states, errs := computeChanges(rules, root)

			require.Len(t, errs, 1)
			assert.Contains(t, errs[0], "マッチ件数 2 が期待値 3 未満")
			assert.Equal(t, dockerfile, states["Dockerfile"].current, "下限未満の rule は内容を書き換えない")
			assert.Empty(t, states["Dockerfile"].applied)
		})

		t.Run("読み取れないファイルは報告して他の rule の処理を続ける", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "Dockerfile", dockerfile)
			rules := []rule{
				dockerfileRule("absent/Dockerfile", "absent", golangFromRe, "1.99.0", 1),
				dockerfileRule("Dockerfile", "golang base", golangFromRe, "1.99.0", 2),
			}

			states, errs := computeChanges(rules, root)

			require.Len(t, errs, 1)
			assert.Contains(t, errs[0], "read absent/Dockerfile")
			assert.Contains(t, states, "Dockerfile")
		})
	})
}

func Test_writeChanges(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("変更のあるファイルを書き出す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, "Dockerfile", "FROM golang:1.26.5-alpine\n")
			states := map[string]*fileState{
				"Dockerfile": {
					original: "FROM golang:1.26.5-alpine\n",
					current:  "FROM golang:1.99.0-alpine\n",
					applied:  []string{"golang base x1"},
				},
			}

			require.NoError(t, writeChanges(states, root))

			data, err := os.ReadFile(filepath.Join(root, "Dockerfile")) //nolint:gosec // path は t.TempDir() 由来
			require.NoError(t, err)
			assert.Equal(t, "FROM golang:1.99.0-alpine\n", string(data))
		})

		t.Run("変更が無いファイルは書き出さない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			states := map[string]*fileState{
				"Dockerfile": {original: "FROM golang:1.26.5-alpine\n", current: "FROM golang:1.26.5-alpine\n"},
			}

			require.NoError(t, writeChanges(states, root))

			_, err := os.Stat(filepath.Join(root, "Dockerfile"))
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("書き込めなければファイル名付きでエラーを返す", func(t *testing.T) {
			t.Parallel()
			states := map[string]*fileState{
				"absent/Dockerfile": {original: "old\n", current: "new\n"},
			}

			err := writeChanges(states, t.TempDir())

			require.ErrorContains(t, err, "write absent/Dockerfile")
		})
	})
}

//nolint:paralleltest // log の出力先を差し替えるため並列化できない
func Test_reportProblems(t *testing.T) {
	t.Run("異常系", func(t *testing.T) {
		//nolint:paralleltest // 親が log の出力先を差し替えるため並列化できない
		t.Run("見出しと理由を 1 件ずつ並べて出力する", func(t *testing.T) {
			out := captureLog(t, func() {
				reportProblems("Validation errors（書き換えは行いません）", []string{
					"go.mod (go directive): version が未設定",
					"docker/tools/Dockerfile (node base): ファイルが見つかりません",
				})
			})

			assert.Contains(t, out, "❌ Validation errors（書き換えは行いません）:")
			assert.Contains(t, out, "  - go.mod (go directive): version が未設定")
			assert.Contains(t, out, "  - docker/tools/Dockerfile (node base): ファイルが見つかりません")
		})
	})
}

func Test_emptyAs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("値があればそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "1.99.0", emptyAs("1.99.0"))
		})

		t.Run("空なら未設定と分かる表示へ置き換える", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "(unset)", emptyAs(""))
		})
	})
}

// writeSyncTargets は、buildRules が挙げる全ファイルを root へ用意します。
// 1 つでも欠けると validateRules が中止させるため、正常系はこの一式が前提になります。
func writeSyncTargets(t *testing.T, root string) {
	t.Helper()

	writeFile(t, root, "mise.toml", fullMiseTOML)
	writeFile(t, root, "go.mod", "module example\n\ngo 0.0.0\n")
	writeFile(t, root, "docker/server/Dockerfile", serverDockerfile)
	writeFile(t, root, "docker/tools/Dockerfile", toolsDockerfile)
	writeFile(t, root, "docker-compose.yaml", composeYAML)

	for _, readme := range []string{
		"docker/README.md", "docker/README.ja.md",
		"docker/server/README.md", "docker/server/README.ja.md",
		"docker/tools/README.md", "docker/tools/README.ja.md",
	} {
		writeFile(t, root, readme, imageReadme)
	}
}

// readSynced は、root からの相対パスのファイル内容を返します。
func readSynced(t *testing.T, root, name string) string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(root, name)) //nolint:gosec // path は t.TempDir と本ファイル内のリテラル
	require.NoError(t, err)

	return string(body)
}

// atRoot は、常に root を返す作業ディレクトリの取得手段です。
func atRoot(root string) func() (string, error) {
	return func() (string, error) { return root, nil }
}

//nolint:paralleltest // run は log へ出力するため、出力先を差し替える他のテストと並行できない
func Test_run(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("mise.toml の版を対象ファイルへ一斉に反映する", func(t *testing.T) {
			root := t.TempDir()
			writeSyncTargets(t, root)

			require.NoError(t, run(atRoot(root)))
			assert.Contains(t, readSynced(t, root, "go.mod"), "go 1.99.0")
			assert.Contains(t, readSynced(t, root, "docker/server/Dockerfile"), "FROM golang:1.99.0-alpine AS builder")
			assert.Contains(t, readSynced(t, root, "docker/tools/Dockerfile"), "FROM node:22.99.0-alpine")
			assert.Contains(t, readSynced(t, root, "docker/tools/Dockerfile"), "FROM python:3.99.0-slim")
			assert.Contains(t, readSynced(t, root, "docker/tools/Dockerfile"), "MISE_VERSION=v2099.1.1")
			assert.Contains(t, readSynced(t, root, "docker-compose.yaml"), "grafana/otel-lgtm:0.99.0")
			assert.Contains(t, readSynced(t, root, "docker/tools/README.md"), "`golang:1.99.0-alpine`")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("作業ディレクトリを取得できなければ伝播元を読みに行かない", func(t *testing.T) {
			failing := func() (string, error) { return "", errFakeGetwd }

			require.ErrorIs(t, run(failing), errFakeGetwd)
		})

		t.Run("mise.toml を読めなければ中止する", func(t *testing.T) {
			root := t.TempDir()

			err := run(atRoot(root))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "mise.toml のパースに失敗")
		})

		t.Run("対象ファイルが 1 つでも欠ければ 1 件も書き換えずに中止する", func(t *testing.T) {
			root := t.TempDir()
			writeSyncTargets(t, root)
			require.NoError(t, os.Remove(filepath.Join(root, "docker/tools/Dockerfile")))

			require.ErrorIs(t, run(atRoot(root)), errAborted)
			assert.Contains(t, readSynced(t, root, "go.mod"), "go 0.0.0")
		})

		t.Run("伝播元に版が無ければ 1 件も書き換えずに中止する", func(t *testing.T) {
			root := t.TempDir()
			writeSyncTargets(t, root)
			writeFile(t, root, "mise.toml", "[tools]\ngo = \"1.99.0\"\n")

			require.ErrorIs(t, run(atRoot(root)), errAborted)
			assert.Contains(t, readSynced(t, root, "go.mod"), "go 0.0.0")
		})

		// 記載が消えたまま通すと、伝播したつもりのファイルが古いままそこに残る。
		t.Run("期待件数に満たない記載があれば 1 件も書き換えずに中止する", func(t *testing.T) {
			root := t.TempDir()
			writeSyncTargets(t, root)
			writeFile(t, root, "docker/server/Dockerfile", "FROM golang:0.0.0-alpine AS base\nARG MISE_VERSION=v0.0.0\n")

			require.ErrorIs(t, run(atRoot(root)), errAborted)
			assert.Contains(t, readSynced(t, root, "go.mod"), "go 0.0.0")
			assert.Contains(t, readSynced(t, root, "docker-compose.yaml"), "grafana/otel-lgtm:0.0.0")
		})
	})
}
