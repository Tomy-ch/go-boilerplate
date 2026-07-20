package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用の 40-hex commit SHA（値は任意、形式のみ意味を持つ）。
const (
	shaCheckout = "1111111111111111111111111111111111111111"
	shaSetupGo  = "2222222222222222222222222222222222222222"
	shaCodeQL   = "3333333333333333333333333333333333333333"
)

func testLock() map[string]string {
	return map[string]string{
		"actions/checkout@v7.0.0": shaCheckout,
		"actions/setup-go@v6":     shaSetupGo,
		"github/codeql-action@v4": shaCodeQL,
	}
}

func TestParseUses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("owner/repo と版から key を構築する", func(t *testing.T) {
			t.Parallel()
			r, ok := parseUses("actions/checkout", "v7.0.0", "")
			require.True(t, ok)
			assert.Equal(t, "actions/checkout", r.repo)
			assert.Empty(t, r.sub)
			assert.Equal(t, "actions/checkout@v7.0.0", r.key())
		})

		t.Run("固定済み行はコメント側の版を採用する", func(t *testing.T) {
			t.Parallel()
			r, ok := parseUses("actions/checkout", shaCheckout, "v7.0.0")
			require.True(t, ok)
			assert.Equal(t, "v7.0.0", r.tag)
			assert.Equal(t, "actions/checkout@v7.0.0", r.key())
		})

		t.Run("サブパスは repo と分離して保持する", func(t *testing.T) {
			t.Parallel()
			r, ok := parseUses("github/codeql-action/init", "v4", "")
			require.True(t, ok)
			assert.Equal(t, "github/codeql-action", r.repo)
			assert.Equal(t, "init", r.sub)
			assert.Equal(t, "github/codeql-action@v4", r.key())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ローカル参照は対象外", func(t *testing.T) {
			t.Parallel()
			_, ok := parseUses("./.github/actions/setup", "", "")
			assert.False(t, ok)
		})

		t.Run("owner/repo に満たない参照は対象外", func(t *testing.T) {
			t.Parallel()
			_, ok := parseUses("single", "v1", "")
			assert.False(t, ok)
		})
	})
}

func TestRewritePins(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録済みで固定済み・一致なら無変更・未登録なし", func(t *testing.T) {
			t.Parallel()
			in := "      - uses: actions/checkout@" + shaCheckout + " # v7.0.0\n"
			out, missing := rewritePins(in, testLock())
			assert.Equal(t, in, out)
			assert.Empty(t, missing)
		})

		t.Run("登録済みで tag のみなら SHA へ固定する", func(t *testing.T) {
			t.Parallel()
			in := "      - uses: actions/setup-go@v6\n"
			out, missing := rewritePins(in, testLock())
			assert.Equal(t, "      - uses: actions/setup-go@"+shaSetupGo+" # v6\n", out)
			assert.Empty(t, missing)
		})

		t.Run("サブパス付きも repo で解決し path を保持する", func(t *testing.T) {
			t.Parallel()
			in := "      - uses: github/codeql-action/init@v4\n"
			out, missing := rewritePins(in, testLock())
			assert.Equal(t, "      - uses: github/codeql-action/init@"+shaCodeQL+" # v4\n", out)
			assert.Empty(t, missing)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SHA 有りでも版が未登録なら未登録として報告する", func(t *testing.T) {
			t.Parallel()
			in := "      - uses: actions/checkout@" + shaCheckout + " # v99.0.0\n"
			out, missing := rewritePins(in, testLock())
			assert.Equal(t, in, out)
			assert.Equal(t, []string{"actions/checkout@v99.0.0"}, missing)
		})

		t.Run("SHA 無しで未登録なら未登録として報告する", func(t *testing.T) {
			t.Parallel()
			in := "      - uses: some/action@v1.2.3\n"
			out, missing := rewritePins(in, testLock())
			assert.Equal(t, in, out)
			assert.Equal(t, []string{"some/action@v1.2.3"}, missing)
		})
	})
}

func TestReadLock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な行のみを repo@tag→SHA として読み込む", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "actions-pin.toml")
			body := "# comment\n" +
				"\"actions/checkout@v7.0.0\" = \"" + shaCheckout + "\"\n" +
				"invalid line\n" +
				"\"actions/setup-go@v6\" = \"" + shaSetupGo + "\"\n"
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

			lock, err := readLock(path)
			require.NoError(t, err)
			assert.Equal(t, map[string]string{
				"actions/checkout@v7.0.0": shaCheckout,
				"actions/setup-go@v6":     shaSetupGo,
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
