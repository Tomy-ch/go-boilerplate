package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用の 64-hex digest（値は任意、形式のみ意味を持つ）。
const (
	digestAlpine = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestGolang = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	digestStale  = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	digestUnreg  = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
)

func testLock() map[string]string {
	return map[string]string{
		"alpine:3.24":        digestAlpine,
		"golang:1.26-alpine": digestGolang,
	}
}

func TestParseRef(t *testing.T) {
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

func TestRewritePins(t *testing.T) {
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

func TestReadLock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な行のみを image:tag→digest として読み込む", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "images-pin.toml")
			body := "# comment\n" +
				"\"alpine:3.24\" = \"" + digestAlpine + "\"\n" +
				"invalid line\n" +
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

		t.Run("ファイルが存在しなければエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := readLock(filepath.Join(t.TempDir(), "absent.toml"))
			require.Error(t, err)
		})
	})
}
