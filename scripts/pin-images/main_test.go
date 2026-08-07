package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用の 64-hex digest（値は任意、形式のみ意味を持つ）。
const (
	digestAlpine = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestGolang = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	digestStale  = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	digestUnreg  = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	digestFresh  = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

// createdLayout は docker buildx imagetools inspect --format が出力する created の形式。
const createdLayout = "2006-01-02 15:04:05 -0700 MST"

// fatalCaseEnv は子プロセスへ渡す実行指示。log.Fatalf は os.Exit するため、非ゼロ終了する
// 経路はテストバイナリ自身を子として再実行して観測する。root は本番と同じく作業ディレクトリから採る。
const fatalCaseEnv = "PIN_IMAGES_TEST_FATAL_CASE"

// errWD は、作業ディレクトリの取得失敗の伝播を検証するためのセンチネルです。
var errWD = xerrors.New("getwd failed")

func TestMain(m *testing.M) {
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}
	switch os.Getenv(fatalCaseEnv) {
	case "":
		os.Exit(m.Run())
	case "fail-on-missing":
		failOnMissing([]string{"busybox:1.36", "busybox:1.36"})
	case "report-drifted":
		report([]string{"docker/app/Dockerfile"}, true, 0)
	case "check-drift":
		applyOrCheck(root, mustTargets(root), true)
	case "apply-missing-no-write":
		applyOrCheck(root, mustTargets(root), false)
	case "resolve-no-stepback":
		resolve(root, mustTargets(root), 14)
	}
	os.Exit(0) // 対象が終了しなかった＝退行。親が非ゼロ終了を期待して落ちる
}

// mustTargets は子プロセス側で targetFiles を呼ぶ。
func mustTargets(root string) []target {
	targets, err := targetFiles(root)
	if err != nil {
		log.Fatalf("targetFiles: %v", err)
	}
	return targets
}

func testLock() map[string]string {
	return map[string]string{
		"alpine:3.24":        digestAlpine,
		"golang:1.26-alpine": digestGolang,
	}
}

// writeFile は親ディレクトリごとファイルを作る。
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
}

// dockerStubBody は、digest 問い合わせには Digest 行を、--format 指定（created 問い合わせ）には
// created を返すダミー docker のシェル本体を組み立てる。
func dockerStubBody(digest string, created time.Time) string {
	return "case \"$*\" in\n" +
		"  *--format*) printf '%s\\n' '" + created.UTC().Format(createdLayout) + "' ;;\n" +
		"  *) printf 'Digest: %s\\n' '" + digest + "' ;;\n" +
		"esac\n"
}

// writeDockerStub は実 docker を呼ばずに inspect の入出力を差し替えるダミーを書き出し、その
// ディレクトリを返す（PATH へは載せない）。
func writeDockerStub(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" + body
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755)) //nolint:gosec // 実行可能スタブ
	return dir
}

// useDockerStub はダミー docker を PATH 先頭へ載せる。
func useDockerStub(t *testing.T, body string) {
	t.Helper()
	t.Setenv("PATH", writeDockerStub(t, body)+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// captureLog は log の出力先を差し替え、書き込まれた内容を読めるようにする。
func captureLog(t *testing.T) *strings.Builder {
	t.Helper()
	var b strings.Builder
	log.SetOutput(&b)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &b
}

// runFatal はテストバイナリ自身を子プロセスとして再実行し、終了コードと出力を返す。
// 子は root を作業ディレクトリとして受け取る。
func runFatal(t *testing.T, name, root, stubDir string) (int, string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), os.Args[0]) //nolint:gosec // 実行するのはテストバイナリ自身
	cmd.Dir = root
	cmd.Env = append(os.Environ(), fatalCaseEnv+"="+name)
	if stubDir != "" {
		cmd.Env = append(cmd.Env, "PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run()
	require.NotNil(t, cmd.ProcessState)
	return cmd.ProcessState.ExitCode(), out.String()
}

func Test_imageRef_key(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("image と tag をコロンで連結する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "golang:1.26-alpine", imageRef{image: "golang", tag: "1.26-alpine"}.key())
		})

		t.Run("registry ポートを含む image でも先頭から連結する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "localhost:5000/app:v2", imageRef{image: "localhost:5000/app", tag: "v2"}.key())
		})
	})
}

func Test_lastColonSplit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最後のコロンで分割する", func(t *testing.T) {
			t.Parallel()
			head, tail, ok := lastColonSplit("localhost:5000/app:v2")
			require.True(t, ok)
			assert.Equal(t, "localhost:5000/app", head)
			assert.Equal(t, "v2", tail)
		})

		t.Run("末尾がコロンなら後半は空になる", func(t *testing.T) {
			t.Parallel()
			head, tail, ok := lastColonSplit("alpine:")
			require.True(t, ok)
			assert.Equal(t, "alpine", head)
			assert.Empty(t, tail)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コロンが無ければ分割しない", func(t *testing.T) {
			t.Parallel()
			head, tail, ok := lastColonSplit("scratch")
			assert.False(t, ok)
			assert.Empty(t, head)
			assert.Empty(t, tail)
		})
	})
}

func Test_parseRef(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("image:tag@digest は digest を捨て image:tag を返す", func(t *testing.T) {
			t.Parallel()
			r, ok := parseRef("golang:1.26-alpine@" + digestGolang)
			require.True(t, ok)
			assert.Equal(t, "golang", r.image)
			assert.Equal(t, "1.26-alpine", r.tag)
			assert.Equal(t, "golang:1.26-alpine", r.key())
		})

		t.Run("registry ホスト付き image も最後のコロンで tag を分離する", func(t *testing.T) {
			t.Parallel()
			r, ok := parseRef("ghcr.io/foo/bar:v1.0")
			require.True(t, ok)
			assert.Equal(t, "ghcr.io/foo/bar", r.image)
			assert.Equal(t, "v1.0", r.tag)
		})

		t.Run("registry ポート付きでも tag を正しく分離する", func(t *testing.T) {
			t.Parallel()
			r, ok := parseRef("localhost:5000/app:v2")
			require.True(t, ok)
			assert.Equal(t, "localhost:5000/app", r.image)
			assert.Equal(t, "v2", r.tag)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("tag 無し（scratch）は対象外", func(t *testing.T) {
			t.Parallel()
			_, ok := parseRef("scratch")
			assert.False(t, ok)
		})

		t.Run("ビルドステージ参照（コロン無し）は対象外", func(t *testing.T) {
			t.Parallel()
			_, ok := parseRef("builder")
			assert.False(t, ok)
		})

		t.Run("最後のコロンが registry:port で tag が無い形は対象外", func(t *testing.T) {
			t.Parallel()
			_, ok := parseRef("localhost:5000/app")
			assert.False(t, ok)
		})
	})
}

func Test_targetFiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Dockerfile には FROM、compose には image 行の正規表現を割り当てる", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "docker", "app", "Dockerfile"), "FROM alpine:3.24\n")
			writeFile(t, filepath.Join(root, "docker-compose.yaml"), "services:\n")

			targets, err := targetFiles(root)
			require.NoError(t, err)
			require.Len(t, targets, 2)

			byPath := map[string]*regexp.Regexp{}
			for _, tg := range targets {
				byPath[tg.path] = tg.re
			}
			assert.Same(t, fromRe, byPath[filepath.Join(root, "docker", "app", "Dockerfile")])
			assert.Same(t, composeImageRe, byPath[filepath.Join(root, "docker-compose.yaml")])
		})

		t.Run("compose は接尾辞違いも収集しパス順に整列する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "docker-compose.yaml"), "services:\n")
			writeFile(t, filepath.Join(root, "docker-compose.attach.yaml"), "services:\n")

			targets, err := targetFiles(root)
			require.NoError(t, err)
			require.Len(t, targets, 2)
			assert.Equal(t, filepath.Join(root, "docker-compose.attach.yaml"), targets[0].path)
			assert.Equal(t, filepath.Join(root, "docker-compose.yaml"), targets[1].path)
		})

		t.Run("走査対象の階層から外れたファイルは含めない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "docker", "app", "nested", "Dockerfile"), "FROM alpine:3.24\n")
			writeFile(t, filepath.Join(root, "docker", "Dockerfile"), "FROM alpine:3.24\n")
			writeFile(t, filepath.Join(root, "sub", "docker-compose.yaml"), "services:\n")
			writeFile(t, filepath.Join(root, "docker", "app", "README.md"), "\n")

			targets, err := targetFiles(root)
			require.NoError(t, err)
			assert.Empty(t, targets)
		})
	})
}

func Test_globFiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("root 配下でパターンに一致したパスだけを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "docker", "app", "Dockerfile"), "")
			writeFile(t, filepath.Join(root, "docker", "app", "README.md"), "")

			files, err := globFiles(root, "docker/*/Dockerfile")

			require.NoError(t, err)
			assert.Equal(t, []string{filepath.Join(root, "docker", "app", "Dockerfile")}, files)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解釈できないパターンは 0 件へ縮退させずエラーを返す", func(t *testing.T) {
			t.Parallel()

			files, err := globFiles(t.TempDir(), "docker/[")

			require.ErrorIs(t, err, filepath.ErrBadPattern)
			assert.Empty(t, files)
		})
	})
}

func Test_collectKeys(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Dockerfile と compose の参照を重複なく集約する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			df := filepath.Join(root, "docker", "app", "Dockerfile")
			cf := filepath.Join(root, "docker-compose.yaml")
			writeFile(t, df, "FROM golang:1.26-alpine AS builder\nFROM alpine:3.24@"+digestAlpine+"\n")
			writeFile(t, cf, "  db:\n    image: alpine:3.24\n")

			keys := collectKeys([]target{{path: df, re: fromRe}, {path: cf, re: composeImageRe}})
			require.Len(t, keys, 2)
			assert.Equal(t, imageRef{image: "alpine", tag: "3.24"}, keys["alpine:3.24"])
			assert.Equal(t, imageRef{image: "golang", tag: "1.26-alpine"}, keys["golang:1.26-alpine"])
		})

		t.Run("scratch とビルドステージ参照は集約しない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			df := filepath.Join(root, "docker", "app", "Dockerfile")
			writeFile(t, df, "FROM scratch\nFROM builder AS final\n")

			assert.Empty(t, collectKeys([]target{{path: df, re: fromRe}}))
		})
	})
}

func Test_uniq(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("隣接する重複を畳む", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"a", "b", "c"}, uniq([]string{"a", "a", "b", "c", "c", "c"}))
		})

		t.Run("重複が無ければ全要素を残す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"a", "b"}, uniq([]string{"a", "b"}))
		})

		t.Run("空スライスは空のまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, uniq(nil))
		})
	})
}

func Test_rel(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("root 配下は root からの相対パスにする", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, filepath.Join("docker", "app", "Dockerfile"),
				rel(filepath.FromSlash("/repo"), filepath.FromSlash("/repo/docker/app/Dockerfile")))
		})

		t.Run("相対化できない組み合わせは元のパスを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "docker/app/Dockerfile", rel(filepath.FromSlash("/repo"), "docker/app/Dockerfile"))
		})
	})
}

func Test_rewritePins(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録済みで digest 固定済み・一致なら無変更・未登録なし", func(t *testing.T) {
			t.Parallel()
			in := "FROM alpine:3.24@" + digestAlpine + " AS base\n"
			out, missing := rewritePins(in, fromRe, testLock())
			assert.Equal(t, in, out)
			assert.Empty(t, missing)
		})

		t.Run("登録済みで tag のみなら digest を付与する（drift）", func(t *testing.T) {
			t.Parallel()
			in := "FROM alpine:3.24\n"
			out, missing := rewritePins(in, fromRe, testLock())
			assert.Equal(t, "FROM alpine:3.24@"+digestAlpine+"\n", out)
			assert.Empty(t, missing)
		})

		t.Run("登録済みで古い digest なら lock の digest へ置換する", func(t *testing.T) {
			t.Parallel()
			in := "FROM alpine:3.24@" + digestStale + "\n"
			out, missing := rewritePins(in, fromRe, testLock())
			assert.Equal(t, "FROM alpine:3.24@"+digestAlpine+"\n", out)
			assert.Empty(t, missing)
		})

		t.Run("AS ステージ名を保持したまま digest を付与する", func(t *testing.T) {
			t.Parallel()
			in := "FROM golang:1.26-alpine AS builder\n"
			out, missing := rewritePins(in, fromRe, testLock())
			assert.Equal(t, "FROM golang:1.26-alpine@"+digestGolang+" AS builder\n", out)
			assert.Empty(t, missing)
		})

		t.Run("scratch とビルドステージ参照は対象外で無変更", func(t *testing.T) {
			t.Parallel()
			in := "FROM scratch\nFROM builder AS final\n"
			out, missing := rewritePins(in, fromRe, testLock())
			assert.Equal(t, in, out)
			assert.Empty(t, missing)
		})

		t.Run("compose の image 行に digest を付与しインデントを保持する", func(t *testing.T) {
			t.Parallel()
			in := "  database:\n    image: alpine:3.24\n"
			out, missing := rewritePins(in, composeImageRe, testLock())
			assert.Equal(t, "  database:\n    image: alpine:3.24@"+digestAlpine+"\n", out)
			assert.Empty(t, missing)
		})

		t.Run("compose の image 行の末尾コメントを保持したまま digest を置換する", func(t *testing.T) {
			t.Parallel()
			in := "    image: alpine:3.24@" + digestStale + " # base\n"
			out, missing := rewritePins(in, composeImageRe, testLock())
			assert.Equal(t, "    image: alpine:3.24@"+digestAlpine+" # base\n", out)
			assert.Empty(t, missing)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("tag のみで未登録なら無変更のまま未登録として報告する", func(t *testing.T) {
			t.Parallel()
			in := "FROM busybox:1.36\n"
			out, missing := rewritePins(in, fromRe, testLock())
			assert.Equal(t, in, out)
			assert.Equal(t, []string{"busybox:1.36"}, missing)
		})

		t.Run("digest 有りでも未登録なら digest を剥がさず未登録として報告する", func(t *testing.T) {
			t.Parallel()
			in := "FROM busybox:1.36@" + digestUnreg + "\n"
			out, missing := rewritePins(in, fromRe, testLock())
			assert.Equal(t, in, out)
			assert.Equal(t, []string{"busybox:1.36"}, missing)
		})

		t.Run("compose の未登録 image は digest を剥がさず未登録として報告する", func(t *testing.T) {
			t.Parallel()
			in := "    image: busybox:1.36@" + digestUnreg + "\n"
			out, missing := rewritePins(in, composeImageRe, testLock())
			assert.Equal(t, in, out)
			assert.Equal(t, []string{"busybox:1.36"}, missing)
		})
	})
}

func Test_readLock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コメントと空行を読み飛ばして image:tag→digest として読み込む", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "images-pin.toml")
			body := "# comment\n" +
				"\n" +
				"\"alpine:3.24\" = \"" + digestAlpine + "\"\n" +
				"\"golang:1.26-alpine\" = \"" + digestGolang + "\"\n"
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

			lock, err := readLock(path)
			require.NoError(t, err)
			assert.Equal(t, map[string]string{
				"alpine:3.24":        digestAlpine,
				"golang:1.26-alpine": digestGolang,
			}, lock)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解釈できない行は読み飛ばさずエラーにする", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "images-pin.toml")
			body := "\"alpine:3.24\" = \"" + digestAlpine + "\"\n" + "invalid line\n"
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

			_, err := readLock(path)

			require.ErrorIs(t, err, errLockInvalidLine)
			assert.ErrorContains(t, err, "2 行目")
		})

		t.Run("キーの重複は後勝ちにせずエラーにする", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "images-pin.toml")
			body := "\"alpine:3.24\" = \"" + digestAlpine + "\"\n" +
				"\"alpine:3.24\" = \"" + digestStale + "\"\n"
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

			_, err := readLock(path)

			require.ErrorIs(t, err, errLockDuplicateKey)
			assert.ErrorContains(t, err, "alpine:3.24")
		})

		t.Run("ファイルが存在しなければエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := readLock(filepath.Join(t.TempDir(), "absent.toml"))
			require.Error(t, err)
		})
	})
}

func Test_writeLock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キー昇順で書き出し readLock で読み戻せる", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "images-pin.toml")
			require.NoError(t, writeLock(path, testLock()))

			body, err := os.ReadFile(path) //nolint:gosec // path は t.TempDir 配下
			require.NoError(t, err)
			assert.Less(t,
				strings.Index(string(body), `"alpine:3.24"`),
				strings.Index(string(body), `"golang:1.26-alpine"`))

			lock, err := readLock(path)
			require.NoError(t, err)
			assert.Equal(t, testLock(), lock)
		})

		t.Run("空の lock でも SSOT の説明ヘッダを持つファイルを作る", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "images-pin.toml")
			require.NoError(t, writeLock(path, map[string]string{}))

			body, err := os.ReadFile(path) //nolint:gosec // path は t.TempDir 配下
			require.NoError(t, err)
			assert.Contains(t, string(body), "pin-images-resolve")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないディレクトリ配下へは書けずエラーを返す", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "absent", "images-pin.toml")
			require.Error(t, writeLock(path, testLock()))
		})
	})
}

func Test_minCreated(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("マルチアーキの複数行から最古を返す", func(t *testing.T) {
			t.Parallel()
			got, ok := minCreated("2026-07-08 01:02:03.4 +0000 UTC\n2026-07-01 00:00:00 +0000 UTC\n")
			require.True(t, ok)
			assert.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), got.UTC())
		})

		t.Run("フィールド数が足りない行と解析できない行は無視する", func(t *testing.T) {
			t.Parallel()
			got, ok := minCreated("\nnot a time\n2026-07-08\n2026-07-08 01:02:03 +0000 UTC\n")
			require.True(t, ok)
			assert.Equal(t, time.Date(2026, time.July, 8, 1, 2, 3, 0, time.UTC), got.UTC())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解析できる行が一つも無ければ見つからないと報告する", func(t *testing.T) {
			t.Parallel()
			_, ok := minCreated("no timestamps here\n")
			assert.False(t, ok)
		})
	})
}

func Test_inspect(t *testing.T) { //nolint:paralleltest // useDockerStub が t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("buildx imagetools inspect へ ref と追加引数をこの順で渡し標準出力を返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "printf '%s' \"$*\"\n")

			out, err := inspect(context.Background(), "alpine:3.24", "--format", "{{ .Name }}")
			require.NoError(t, err)
			assert.Equal(t, "buildx imagetools inspect alpine:3.24 --format {{ .Name }}", out)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("docker が非 0 終了なら ref と標準エラー出力を含むエラーを返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "echo 'toomanyrequests' >&2\nexit 1\n")

			_, err := inspect(context.Background(), "alpine:3.24")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "alpine:3.24")
			assert.Contains(t, err.Error(), "toomanyrequests")
		})
	})
}

func Test_resolveDigest(t *testing.T) { //nolint:paralleltest // useDockerStub が t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("inspect 出力の Digest 行から digest を取り出す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "printf 'Name: alpine:3.24\\nDigest: %s\\n' '"+digestAlpine+"'\n")

			got, err := resolveDigest(context.Background(), "alpine:3.24")
			require.NoError(t, err)
			assert.Equal(t, digestAlpine, got)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("Digest 行が無ければ解析不能として扱う", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "printf 'Name: alpine:3.24\\n'\n")

			_, err := resolveDigest(context.Background(), "alpine:3.24")
			require.ErrorIs(t, err, errDigestUnparsable)
		})

		t.Run("inspect が失敗すればそのエラーを伝播する", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "exit 1\n")

			_, err := resolveDigest(context.Background(), "alpine:3.24")
			require.Error(t, err)
			require.NotErrorIs(t, err, errDigestUnparsable)
		})
	})
}

func Test_earliestCreated(t *testing.T) { //nolint:paralleltest // useDockerStub が t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("マルチアーキは全アーキの created のうち最古を返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "printf '2026-07-08 01:02:03 +0000 UTC\\n2026-07-01 00:00:00 +0000 UTC\\n'\n")

			got, err := earliestCreated(context.Background(), "alpine:3.24")
			require.NoError(t, err)
			assert.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), got.UTC())
		})

		t.Run("index 用テンプレートで解析できなければ単一アーキ用へフォールバックする", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "case \"$*\" in\n"+
				"  *range*) exit 0 ;;\n"+
				"esac\n"+
				"printf '2026-07-08 01:02:03 +0000 UTC\\n'\n")

			got, err := earliestCreated(context.Background(), "alpine:3.24")
			require.NoError(t, err)
			assert.Equal(t, time.Date(2026, time.July, 8, 1, 2, 3, 0, time.UTC), got.UTC())
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("inspect 自体が失敗し続ければ解析不能ではなくその失敗を返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "echo 'toomanyrequests' >&2\nexit 1\n")

			_, err := earliestCreated(context.Background(), "alpine:3.24")
			require.Error(t, err)
			require.NotErrorIs(t, err, errCreatedUnparsable)
			assert.Contains(t, err.Error(), "toomanyrequests")
		})

		t.Run("どちらのテンプレートでも created を読めなければ解析不能として扱う", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "printf 'no timestamps here\\n'\n")

			_, err := earliestCreated(context.Background(), "alpine:3.24")
			require.ErrorIs(t, err, errCreatedUnparsable)
		})
	})
}

func Test_digestAgeDays(t *testing.T) { //nolint:paralleltest // useDockerStub が t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("created からの経過を日数へ丸めて返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			created := time.Now().UTC().Add(-30*hoursPerDay*time.Hour - time.Hour)
			useDockerStub(t, dockerStubBody(digestAlpine, created))

			got, err := digestAgeDays(context.Background(), "alpine:3.24")
			require.NoError(t, err)
			assert.Equal(t, 30, got)
		})

		t.Run("公開直後は 0 日として返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, dockerStubBody(digestAlpine, time.Now().UTC().Add(-time.Hour)))

			got, err := digestAgeDays(context.Background(), "alpine:3.24")
			require.NoError(t, err)
			assert.Zero(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("created を取得できなければエラーを返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, "exit 1\n")

			_, err := digestAgeDays(context.Background(), "alpine:3.24")
			require.Error(t, err)
		})
	})
}

func Test_quarantine(t *testing.T) { //nolint:paralleltest // useDockerStub が t.Setenv を使うため並列化不可
	ref := imageRef{image: "alpine", tag: "3.24"}
	key := ref.key()
	existing := map[string]string{key: digestStale}

	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("cooldown 無効なら age を問わず現 digest を採用する", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			use, note := quarantine(context.Background(), ref, key, digestFresh, 0, existing)
			assert.Equal(t, digestFresh, use)
			assert.Empty(t, note)
		})

		t.Run("窓を越えた digest はそのまま採用する", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, dockerStubBody(digestFresh, time.Now().UTC().Add(-30*hoursPerDay*time.Hour)))

			use, note := quarantine(context.Background(), ref, key, digestFresh, 14, existing)
			assert.Equal(t, digestFresh, use)
			assert.Empty(t, note)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("窓の内側で既存ピンがあれば出来立てを採らず前回 lock へ退く", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, dockerStubBody(digestFresh, time.Now().UTC()))

			use, note := quarantine(context.Background(), ref, key, digestFresh, 14, existing)
			assert.Equal(t, digestStale, use)
			assert.Contains(t, note, key)
			assert.Contains(t, note, "既存ピンを維持")
		})

		t.Run("窓の内側で既存ピンが無ければ採用せず skip する", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useDockerStub(t, dockerStubBody(digestFresh, time.Now().UTC()))

			use, note := quarantine(context.Background(), ref, key, digestFresh, 14, map[string]string{})
			assert.Empty(t, use)
			assert.Contains(t, note, "skip")
		})
	})
}

func Test_resolve(t *testing.T) { //nolint:paralleltest // useDockerStub が t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("収集した image の digest を解決し lockfile へ書き出す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "docker", "app", "Dockerfile"), "FROM alpine:3.24\n")
			useDockerStub(t, dockerStubBody(digestAlpine, time.Now().UTC()))

			resolve(root, mustTargets(root), 0)

			lock, err := readLock(filepath.Join(root, lockFile))
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"alpine:3.24": digestAlpine}, lock)
		})

		t.Run("窓の内側なら既存 lockfile の digest を維持して書き出す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "docker", "app", "Dockerfile"), "FROM alpine:3.24\n")
			writeFile(t, filepath.Join(root, lockFile), "\"alpine:3.24\" = \""+digestStale+"\"\n")
			useDockerStub(t, dockerStubBody(digestFresh, time.Now().UTC()))

			resolve(root, mustTargets(root), 14)

			lock, err := readLock(filepath.Join(root, lockFile))
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"alpine:3.24": digestStale}, lock)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("退行先の無い出来立て image は lockfile へ載せず非ゼロ終了する", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "docker", "app", "Dockerfile"), "FROM alpine:3.24\n")
			stub := writeDockerStub(t, dockerStubBody(digestFresh, time.Now().UTC()))

			code, out := runFatal(t, "resolve-no-stepback", root, stub)
			assert.NotZero(t, code)
			assert.Contains(t, out, "退行先の無い出来立て image")

			body, err := os.ReadFile(filepath.Join(root, lockFile)) //nolint:gosec // path は t.TempDir 配下
			require.NoError(t, err)
			assert.NotContains(t, string(body), digestFresh)
		})
	})
}

func Test_applyOrCheck(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lockfile の digest で Dockerfile と compose を固定する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			df := filepath.Join(root, "docker", "app", "Dockerfile")
			cf := filepath.Join(root, "docker-compose.yaml")
			writeFile(t, df, "FROM golang:1.26-alpine AS builder\n")
			writeFile(t, cf, "  db:\n    image: alpine:3.24\n")
			require.NoError(t, writeLock(filepath.Join(root, lockFile), testLock()))

			applyOrCheck(root, mustTargets(root), false)

			assert.Equal(t, "FROM golang:1.26-alpine@"+digestGolang+" AS builder\n", readAll(t, df))
			assert.Equal(t, "  db:\n    image: alpine:3.24@"+digestAlpine+"\n", readAll(t, cf))
		})

		t.Run("固定済みなら check は失敗せずファイルも書き換えない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			df := filepath.Join(root, "docker", "app", "Dockerfile")
			body := "FROM alpine:3.24@" + digestAlpine + "\n"
			writeFile(t, df, body)
			require.NoError(t, writeLock(filepath.Join(root, lockFile), testLock()))

			applyOrCheck(root, mustTargets(root), true)

			assert.Equal(t, body, readAll(t, df))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未固定のまま check すると書き換えずに非ゼロ終了する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			df := filepath.Join(root, "docker", "app", "Dockerfile")
			writeFile(t, df, "FROM alpine:3.24\n")
			require.NoError(t, writeLock(filepath.Join(root, lockFile), testLock()))

			code, out := runFatal(t, "check-drift", root, "")
			assert.NotZero(t, code)
			assert.Contains(t, out, filepath.Join("docker", "app", "Dockerfile"))
			assert.Equal(t, "FROM alpine:3.24\n", readAll(t, df))
		})

		t.Run("後続ファイルに未登録があれば先行ファイルも書き換えずに非ゼロ終了する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			df := filepath.Join(root, "docker", "app", "Dockerfile")
			cf := filepath.Join(root, "docker-compose.yaml")
			writeFile(t, df, "FROM alpine:3.24\n")
			writeFile(t, cf, "  db:\n    image: busybox:1.36\n")
			require.NoError(t, writeLock(filepath.Join(root, lockFile), testLock()))

			code, out := runFatal(t, "apply-missing-no-write", root, "")

			assert.NotZero(t, code)
			assert.Contains(t, out, "busybox:1.36")
			assert.Equal(t, "FROM alpine:3.24\n", readAll(t, df))
			assert.Equal(t, "  db:\n    image: busybox:1.36\n", readAll(t, cf))
		})
	})
}

func Test_report(t *testing.T) { //nolint:paralleltest // captureLog が log の出力先を差し替えるため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
		t.Run("apply では固定したファイル数を報告する", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			out := captureLog(t)

			report(nil, false, 3)

			assert.Contains(t, out.String(), "3 ファイルを固定しました")
		})

		t.Run("check では未固定も未登録も無いことを報告する", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			out := captureLog(t)

			report(nil, true, 0)

			assert.Contains(t, out.String(), "全 base image が lockfile 通りに固定されています")
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
		t.Run("check で drift を見つけたら非ゼロ終了する", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			code, out := runFatal(t, "report-drifted", "", "")
			assert.NotZero(t, code)
			assert.Contains(t, out, "docker/app/Dockerfile")
		})
	})
}

func Test_failOnMissing(t *testing.T) { //nolint:paralleltest // 子プロセス再実行が log の出力先を共有するため
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
		t.Run("未登録が無ければ何も出力せず処理を続けさせる", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			out := captureLog(t)

			failOnMissing(nil)

			assert.Empty(t, out.String())
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
		t.Run("lockfile 未登録があれば重複を畳んで非ゼロ終了する", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			code, out := runFatal(t, "fail-on-missing", "", "")
			assert.NotZero(t, code)
			assert.Contains(t, out, "未登録")
			assert.Equal(t, 1, strings.Count(out, "busybox:1.36"))
		})
	})
}

// readAll はテスト対象が書き換えたファイルの内容を返す。
func readAll(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // path は t.TempDir 配下
	require.NoError(t, err)
	return string(b)
}

// stubWD は、固定のディレクトリを返す作業ディレクトリの取得手段です。
func stubWD(root string) func() (string, error) {
	return func() (string, error) { return root, nil }
}

func Test_run(t *testing.T) {
	t.Parallel()

	lockBody := "\"alpine:3.24\" = \"" + digestAlpine + "\"\n"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("resolve は走査結果で lockfile を書き直す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, lockFile), lockBody)

			require.NoError(t, run([]string{"resolve"}, stubWD(root)))

			lock, err := readLock(filepath.Join(root, lockFile))
			require.NoError(t, err)
			assert.Empty(t, lock, "resolve 以外へ振り分けると lockfile が据え置かれる")
		})

		t.Run("apply は lockfile の digest でファイルを固定する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			df := filepath.Join(root, "docker", "app", "Dockerfile")
			writeFile(t, df, "FROM alpine:3.24\n")
			writeFile(t, filepath.Join(root, lockFile), lockBody)

			require.NoError(t, run([]string{"apply"}, stubWD(root)))

			assert.Equal(t, "FROM alpine:3.24@"+digestAlpine+"\n", readAll(t, df))
		})

		t.Run("check は作業ツリーも lockfile も書き換えない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			df := filepath.Join(root, "docker", "app", "Dockerfile")
			body := "FROM alpine:3.24@" + digestAlpine + "\n"
			writeFile(t, df, body)
			writeFile(t, filepath.Join(root, lockFile), lockBody)

			require.NoError(t, run([]string{"check"}, stubWD(root)))

			assert.Equal(t, body, readAll(t, df))
			assert.Equal(t, lockBody, readAll(t, filepath.Join(root, lockFile)),
				"resolve へ振り分けると lockfile が書き直される")
		})

		t.Run("ヘルプ要求は失敗にしない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, lockFile), lockBody)

			require.NoError(t, run([]string{"resolve", "-h"}, stubWD(root)))

			assert.Equal(t, lockBody, readAll(t, filepath.Join(root, lockFile)),
				"ヘルプ要求で resolve まで進むと lockfile が書き直される")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブコマンドが無ければ使い方を返す", func(t *testing.T) {
			t.Parallel()

			err := run(nil, stubWD(t.TempDir()))

			require.ErrorIs(t, err, errUsage)
		})

		t.Run("未知のサブコマンドは使い方を返す", func(t *testing.T) {
			t.Parallel()

			err := run([]string{"bogus"}, stubWD(t.TempDir()))

			require.ErrorIs(t, err, errUsage)
		})

		t.Run("作業ディレクトリを取得できなければ失敗する", func(t *testing.T) {
			t.Parallel()

			err := run([]string{"check"}, func() (string, error) { return "", errWD })

			require.ErrorIs(t, err, errWD)
			assert.ErrorContains(t, err, "getwd")
		})

		t.Run("走査対象を集められなければ失敗する", func(t *testing.T) {
			t.Parallel()

			err := run([]string{"check"}, stubWD(filepath.Join(t.TempDir(), "x[")))

			require.ErrorIs(t, err, filepath.ErrBadPattern)
		})

		t.Run("未知のフラグはヘルプ要求と混同せず失敗する", func(t *testing.T) {
			t.Parallel()

			err := run([]string{"resolve", "-bogus"}, stubWD(t.TempDir()))

			require.ErrorContains(t, err, "failed to parse flags")
		})
	})
}
