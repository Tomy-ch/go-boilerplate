package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用の 40-hex commit SHA（値は任意、形式のみ意味を持つ）。
const (
	shaCheckout = "1111111111111111111111111111111111111111"
	shaSetupGo  = "2222222222222222222222222222222222222222"
	shaCodeQL   = "3333333333333333333333333333333333333333"
)

// errAge は、ageFn の失敗伝播を検証するためのセンチネルです。
var errAge = xerrors.New("age lookup failed")

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

		t.Run("docker:// 参照は無意味な repo を作らず対象外", func(t *testing.T) {
			t.Parallel()
			_, ok := parseUses("docker://alpine", "3.22", "")
			assert.False(t, ok)
		})

		t.Run("owner を持つ docker:// 参照も対象外", func(t *testing.T) {
			t.Parallel()
			_, ok := parseUses("docker://ghcr.io/owner/image", "1.0.0", "")
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

		t.Run("コメント行と空行は読み飛ばして repo@tag→SHA を読み込む", func(t *testing.T) {
			t.Parallel()
			body := "# comment\n" +
				"\n" +
				"\"actions/checkout@v7.0.0\" = \"" + shaCheckout + "\"\n" +
				"\"actions/setup-go@v6\" = \"" + shaSetupGo + "\"\n"

			lock, err := readLock(writeLockFile(t, body))

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

		t.Run("代入として解釈できない行があれば行番号付きでエラーを返す", func(t *testing.T) {
			t.Parallel()
			body := "\"actions/checkout@v7.0.0\" = \"" + shaCheckout + "\"\n" +
				"invalid line\n"

			_, err := readLock(writeLockFile(t, body))

			require.ErrorIs(t, err, errLockInvalidLine)
			require.ErrorContains(t, err, "2 行目")
		})

		t.Run("先頭行がいきなり不正でも行番号を 1 と報告する", func(t *testing.T) {
			t.Parallel()

			_, err := readLock(writeLockFile(t, "invalid line\n"))

			require.ErrorIs(t, err, errLockInvalidLine)
			require.ErrorContains(t, err, "1 行目")
		})

		t.Run("キーが重複していれば後勝ちにせずエラーを返す", func(t *testing.T) {
			t.Parallel()
			body := "\"actions/checkout@v7.0.0\" = \"" + shaCheckout + "\"\n" +
				"\"actions/checkout@v7.0.0\" = \"" + shaSetupGo + "\"\n"

			_, err := readLock(writeLockFile(t, body))

			require.ErrorIs(t, err, errLockDuplicateKey)
			require.ErrorContains(t, err, "2 行目")
			require.ErrorContains(t, err, "actions/checkout@v7.0.0")
		})
	})
}

// writeLockFile は body を lockfile として一時ディレクトリへ書き出し、そのパスを返す。
func writeLockFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "actions-pin.toml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestQuarantine(t *testing.T) {
	t.Parallel()

	const (
		key       = "actions/checkout@v7.0.0"
		candidate = shaCheckout
		prev      = "9999999999999999999999999999999999999999"
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("minAgeDays<=0 なら ageFn を呼ばず候補をそのまま採用する", func(t *testing.T) {
			t.Parallel()
			called := false
			ageFn := func() (int, error) { called = true; return 0, nil }

			use, note, err := quarantine(ageFn, key, candidate, 0, nil)

			require.NoError(t, err)
			assert.Equal(t, candidate, use)
			assert.Empty(t, note)
			assert.False(t, called, "minAgeDays<=0 では ageFn を呼ばない")
		})

		t.Run("解決先が十分に古ければ候補を採用する", func(t *testing.T) {
			t.Parallel()
			ageFn := func() (int, error) { return 30, nil }

			use, note, err := quarantine(ageFn, key, candidate, 14, nil)

			require.NoError(t, err)
			assert.Equal(t, candidate, use)
			assert.Empty(t, note)
		})

		t.Run("新しすぎる場合は既存ピンを維持しノートを返す", func(t *testing.T) {
			t.Parallel()
			ageFn := func() (int, error) { return 3, nil }
			existing := map[string]string{key: prev}

			use, note, err := quarantine(ageFn, key, candidate, 14, existing)

			require.NoError(t, err)
			assert.Equal(t, prev, use)
			assert.Contains(t, note, "既存ピンを維持")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("新しすぎて既存ピンも無ければ skip しノートを返す", func(t *testing.T) {
			t.Parallel()
			ageFn := func() (int, error) { return 3, nil }

			use, note, err := quarantine(ageFn, key, candidate, 14, nil)

			require.NoError(t, err)
			assert.Empty(t, use)
			assert.Contains(t, note, "skip")
		})

		t.Run("ageFn の失敗はそのまま伝播する", func(t *testing.T) {
			t.Parallel()
			ageFn := func() (int, error) { return 0, errAge }

			use, note, err := quarantine(ageFn, key, candidate, 14, nil)

			require.ErrorIs(t, err, errAge)
			assert.Empty(t, use)
			assert.Empty(t, note)
		})
	})
}

func TestSelectSHA(t *testing.T) {
	t.Parallel()

	const (
		tag         = "v7.0.0"
		derefSHA    = shaCheckout
		lightTagSHA = shaSetupGo
		headSHA     = shaCodeQL
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("annotated tag の deref を最優先で採用する", func(t *testing.T) {
			t.Parallel()
			out := lightTagSHA + "\trefs/tags/" + tag + "\n" +
				derefSHA + "\trefs/tags/" + tag + "^{}\n"
			sha, err := selectSHA(out, tag)
			require.NoError(t, err)
			assert.Equal(t, derefSHA, sha)
		})

		t.Run("deref が無ければ軽量 tag を採用する", func(t *testing.T) {
			t.Parallel()
			out := lightTagSHA + "\trefs/tags/" + tag + "\n"
			sha, err := selectSHA(out, tag)
			require.NoError(t, err)
			assert.Equal(t, lightTagSHA, sha)
		})

		t.Run("tag が無ければ branch head へフォールバックする", func(t *testing.T) {
			t.Parallel()
			out := headSHA + "\trefs/heads/" + tag + "\n"
			sha, err := selectSHA(out, tag)
			require.NoError(t, err)
			assert.Equal(t, headSHA, sha)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("該当 ref が無ければ未発見エラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := selectSHA("", tag)
			require.ErrorIs(t, err, errRefNotFound)
			require.ErrorContains(t, err, tag)
		})

		t.Run("無関係な ref のみなら未発見エラーを返す", func(t *testing.T) {
			t.Parallel()
			out := headSHA + "\trefs/heads/main\n"
			_, err := selectSHA(out, tag)
			require.ErrorIs(t, err, errRefNotFound)
		})
	})
}

func TestIsIgnorableLockErr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nil は無視可能", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isIgnorableLockErr(nil))
		})

		t.Run("ファイル不在は無視可能", func(t *testing.T) {
			t.Parallel()
			_, err := os.Open(filepath.Join(t.TempDir(), "absent.toml"))
			require.Error(t, err)
			assert.True(t, isIgnorableLockErr(err))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不在以外のエラーは無視不可（fail-close）", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isIgnorableLockErr(errAge))
		})
	})
}

func TestPickRefTime(t *testing.T) {
	t.Parallel()

	var (
		older = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
		newer = time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Release の方が新しければ published_at を採用する", func(t *testing.T) {
			t.Parallel()
			got, err := pickRefTime(newer, older)
			require.NoError(t, err)
			assert.Equal(t, newer, got)
		})

		t.Run("tag 付け替えで commit の方が新しければ commit 日時を採用する", func(t *testing.T) {
			t.Parallel()
			got, err := pickRefTime(older, newer)
			require.NoError(t, err)
			assert.Equal(t, newer, got)
		})

		t.Run("両者が同一時刻ならその時刻を返す", func(t *testing.T) {
			t.Parallel()
			got, err := pickRefTime(older, older)
			require.NoError(t, err)
			assert.Equal(t, older, got)
		})

		t.Run("Release が無ければ commit 日時を採用する", func(t *testing.T) {
			t.Parallel()
			got, err := pickRefTime(time.Time{}, older)
			require.NoError(t, err)
			assert.Equal(t, older, got)
		})

		t.Run("commit 日時が無ければ published_at を採用する", func(t *testing.T) {
			t.Parallel()
			got, err := pickRefTime(older, time.Time{})
			require.NoError(t, err)
			assert.Equal(t, older, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("どちらの日時も得られなければエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := pickRefTime(time.Time{}, time.Time{})
			require.ErrorIs(t, err, errRefDateUnavailable)
		})
	})
}

func TestTargetFiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("workflows の yml と yaml を対象にする", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, ".github/workflows/a.yml", "")
			writeFile(t, root, ".github/workflows/b.yaml", "")

			files, err := targetFiles(root)

			require.NoError(t, err)
			assert.Equal(t, []string{
				filepath.Join(root, ".github/workflows/a.yml"),
				filepath.Join(root, ".github/workflows/b.yaml"),
			}, files)
		})

		t.Run("入れ子の composite action も対象にする", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, ".github/actions/flat/action.yml", "")
			writeFile(t, root, ".github/actions/group/nested/action.yaml", "")

			files, err := targetFiles(root)

			require.NoError(t, err)
			assert.Equal(t, []string{
				filepath.Join(root, ".github/actions/flat/action.yml"),
				filepath.Join(root, ".github/actions/group/nested/action.yaml"),
			}, files)
		})

		t.Run("action.yml 以外のファイルは対象にしない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, ".github/actions/flat/action.yml", "")
			writeFile(t, root, ".github/actions/flat/README.md", "")
			writeFile(t, root, ".github/actions/flat/helper.yaml", "")

			files, err := targetFiles(root)

			require.NoError(t, err)
			assert.Equal(t, []string{filepath.Join(root, ".github/actions/flat/action.yml")}, files)
		})

		t.Run("actions ディレクトリが無ければ workflows のみを対象にする", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, ".github/workflows/a.yml", "")

			files, err := targetFiles(root)

			require.NoError(t, err)
			assert.Equal(t, []string{filepath.Join(root, ".github/workflows/a.yml")}, files)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不在以外の走査エラーは握り潰さずエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, root, ".github", "")

			_, err := targetFiles(root)

			require.Error(t, err)
			assert.False(t, xerrors.Is(err, os.ErrNotExist), "不在エラーとして無視されていない")
		})
	})
}

func TestFileRefs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一ファイル内の複数の uses: を出現順に返す", func(t *testing.T) {
			t.Parallel()
			data := "      - uses: actions/checkout@v7.0.0\n" +
				"      - uses: actions/setup-go@v6\n"

			refs := fileRefs(data)

			require.Len(t, refs, 2)
			assert.Equal(t, "actions/checkout@v7.0.0", refs[0].key())
			assert.Equal(t, "actions/setup-go@v6", refs[1].key())
		})

		t.Run("ローカル参照は含めない", func(t *testing.T) {
			t.Parallel()
			refs := fileRefs("      - uses: ./.github/actions/setup-postgres\n")
			assert.Empty(t, refs)
		})

		t.Run("docker:// 参照は壊れたキーを作らず含めない", func(t *testing.T) {
			t.Parallel()
			refs := fileRefs("      - uses: docker://alpine@sha256:0000000000000000000000000000000000000000000000000000000000000000\n")
			assert.Empty(t, refs)
		})

		t.Run("クオートされたブロック記法は壊れたキーを作らず含めない", func(t *testing.T) {
			t.Parallel()
			refs := fileRefs("      - uses: 'actions/checkout@v7.0.0'\n")
			assert.Empty(t, refs)
		})
	})
}

func TestDetectLooseUses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ブロック記法は厳密パターンで処理済みとして検出しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, detectLooseUses("      - uses: actions/checkout@v7.0.0\n"))
		})

		t.Run("固定済みのブロック記法も検出しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, detectLooseUses("      - uses: actions/checkout@"+shaCheckout+" # v7.0.0\n"))
		})

		t.Run("クオートされたブロック記法は解釈できない記法として検出する", func(t *testing.T) {
			t.Parallel()
			found := detectLooseUses("      - uses: 'actions/checkout@v7.0.0'\n")
			assert.Equal(t, []string{"actions/checkout@v7.0.0"}, found)
		})

		t.Run("docker:// 参照は厳密パターンで処理済みとして検出しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, detectLooseUses("      - uses: docker://alpine@sha256:0000000000000000000000000000000000000000000000000000000000000000\n"))
		})

		t.Run("flow mapping でもローカル参照は検出しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, detectLooseUses("      - {name: Setup, uses: ./.github/actions/setup-postgres}\n"))
		})

		t.Run("版を持たない参照は検出しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, detectLooseUses("      - {name: Checkout, uses: actions/checkout}\n"))
		})

		t.Run("コメント行の記述例は検出しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, detectLooseUses("      # uses: actions/checkout@v7.0.0 のように書く\n"))
		})

		t.Run("run: スクリプト中の uses: を含む文字列は検出しない", func(t *testing.T) {
			t.Parallel()
			data := "      - run: |\n" +
				"          echo \"this workflow uses: actions/checkout@v7.0.0 internally\"\n"
			assert.Empty(t, detectLooseUses(data))
		})

		t.Run("ブロックスカラーを抜けた後の行は再び走査対象に戻る", func(t *testing.T) {
			t.Parallel()
			data := "      - run: |\n" +
				"          echo hi\n" +
				"      - {uses: actions/checkout@v7.0.0}\n"
			assert.Equal(t, []string{"actions/checkout@v7.0.0"}, detectLooseUses(data))
		})

		t.Run("uses で終わる別のキーは検出しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, detectLooseUses("      disuses: actions/checkout@v7.0.0\n"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("flow mapping 記法の外部参照を検出する", func(t *testing.T) {
			t.Parallel()
			got := detectLooseUses("      - {name: Checkout, uses: actions/checkout@v7.0.0}\n")
			assert.Equal(t, []string{"actions/checkout@v7.0.0"}, got)
		})

		t.Run("クオート付きの外部参照も検出する", func(t *testing.T) {
			t.Parallel()
			got := detectLooseUses("      - {name: Checkout, uses: \"actions/checkout@v7.0.0\"}\n")
			assert.Equal(t, []string{"actions/checkout@v7.0.0"}, got)
		})

		t.Run("サブパス付きの外部参照も検出する", func(t *testing.T) {
			t.Parallel()
			got := detectLooseUses("      - {name: Init, uses: github/codeql-action/init@v4}\n")
			assert.Equal(t, []string{"github/codeql-action/init@v4"}, got)
		})

		t.Run("クオートしたキーの外部参照も検出する", func(t *testing.T) {
			t.Parallel()
			got := detectLooseUses("      - name: Checkout\n        \"uses\": actions/checkout@v7.0.0\n")
			assert.Equal(t, []string{"actions/checkout@v7.0.0"}, got)
		})

		t.Run("折り畳みブロックスカラーで値を次行へ送った uses: を検出する", func(t *testing.T) {
			t.Parallel()
			got := detectLooseUses("      - uses: >-\n          actions/checkout@v7.0.0\n")
			assert.Equal(t, []string{"uses: の値が同じ行にありません"}, got)
		})

		t.Run("リテラルブロックスカラーで値を次行へ送った uses: を検出する", func(t *testing.T) {
			t.Parallel()
			got := detectLooseUses("      - uses: |-\n          actions/checkout@v7.0.0\n")
			assert.Equal(t, []string{"uses: の値が同じ行にありません"}, got)
		})

		t.Run("YAML alias の uses: は解決できないものとして検出する", func(t *testing.T) {
			t.Parallel()
			got := detectLooseUses("      - uses: *checkout\n")
			require.Len(t, got, 1)
			assert.Contains(t, got[0], "alias")
		})

		t.Run("同一参照が複数あっても重複を除いて返す", func(t *testing.T) {
			t.Parallel()
			data := "      - {uses: actions/checkout@v7.0.0}\n" +
				"      - {uses: actions/checkout@v7.0.0}\n"
			got := detectLooseUses(data)
			assert.Equal(t, []string{"actions/checkout@v7.0.0"}, got)
		})
	})
}

func TestOrphanKeys(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全キーが参照されていれば孤児は無い", func(t *testing.T) {
			t.Parallel()
			used := map[string]bool{
				"actions/checkout@v7.0.0": true,
				"actions/setup-go@v6":     true,
				"github/codeql-action@v4": true,
			}
			assert.Empty(t, orphanKeys(testLock(), used))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("どの uses: からも参照されないキーを昇順で返す", func(t *testing.T) {
			t.Parallel()
			used := map[string]bool{"actions/setup-go@v6": true}

			got := orphanKeys(testLock(), used)

			assert.Equal(t, []string{"actions/checkout@v7.0.0", "github/codeql-action@v4"}, got)
		})
	})
}

func TestPlanRewrites(t *testing.T) {
	t.Parallel()

	const (
		resolvable   = "      - uses: actions/setup-go@v6\n"
		unregistered = "      - uses: some/action@v1.2.3\n"
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解決可能なファイルの固定後の内容を計画に載せる", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := writeFile(t, root, ".github/workflows/a.yml", resolvable)

			plan, err := planRewrites(root, []string{path}, testLock())

			require.NoError(t, err)
			require.NoError(t, plan.validate(map[string]string{"actions/setup-go@v6": shaSetupGo}))
			assert.Equal(t, map[string]string{
				path: "      - uses: actions/setup-go@" + shaSetupGo + " # v6\n",
			}, plan.changes)
		})

		t.Run("固定済みで一致するファイルは変更対象に含めない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			body := "      - uses: actions/setup-go@" + shaSetupGo + " # v6\n"
			path := writeFile(t, root, ".github/workflows/a.yml", body)

			plan, err := planRewrites(root, []string{path}, testLock())

			require.NoError(t, err)
			assert.Empty(t, plan.changes)
		})

		t.Run("参照された lockfile のキーを used として記録する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := writeFile(t, root, ".github/workflows/a.yml", resolvable)

			plan, err := planRewrites(root, []string{path}, testLock())

			require.NoError(t, err)
			assert.Equal(t, map[string]bool{"actions/setup-go@v6": true}, plan.used)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録参照が混在しても計画段階ではファイルを書き換えない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			body := resolvable + unregistered
			path := writeFile(t, root, ".github/workflows/a.yml", body)

			plan, err := planRewrites(root, []string{path}, testLock())

			require.NoError(t, err)
			require.ErrorIs(t, plan.validate(testLock()), errLockMissingKey)
			onDisk, err := os.ReadFile(path) //nolint:gosec // path は t.TempDir() 由来
			require.NoError(t, err)
			assert.Equal(t, body, string(onDisk), "中断時に作業ツリーが書き換わっていない")
		})

		t.Run("未登録参照は他ファイルの解決可能な変更より優先して報告する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ok := writeFile(t, root, ".github/workflows/a.yml", resolvable)
			ng := writeFile(t, root, ".github/workflows/b.yml", unregistered)

			plan, err := planRewrites(root, []string{ok, ng}, testLock())

			require.NoError(t, err)
			require.ErrorIs(t, plan.validate(testLock()), errLockMissingKey)
			assert.Contains(t, plan.changes, ok, "解決可能なファイルの計画自体は保持する")
		})

		t.Run("厳密パターンで解釈できない記法があれば未登録より先に報告する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			body := unregistered + "      - {name: Checkout, uses: actions/checkout@v7.0.0}\n"
			path := writeFile(t, root, ".github/workflows/a.yml", body)

			plan, err := planRewrites(root, []string{path}, testLock())

			require.NoError(t, err)
			err = plan.validate(testLock())
			require.ErrorIs(t, err, errLooseUses)
			require.ErrorContains(t, err, ".github/workflows/a.yml")
		})

		t.Run("参照されない lockfile エントリがあれば孤児として報告する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			body := "      - uses: actions/setup-go@" + shaSetupGo + " # v6\n"
			path := writeFile(t, root, ".github/workflows/a.yml", body)

			plan, err := planRewrites(root, []string{path}, testLock())

			require.NoError(t, err)
			err = plan.validate(testLock())
			require.ErrorIs(t, err, errLockOrphanKey)
			require.ErrorContains(t, err, "actions/checkout@v7.0.0")
		})

		t.Run("読み取れないファイルがあればエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()

			_, err := planRewrites(root, []string{filepath.Join(root, "absent.yml")}, testLock())

			require.ErrorContains(t, err, "absent.yml")
		})
	})
}

// writeFile は root からの相対パスへ body を書き出し、その絶対パスを返す。
func writeFile(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}
