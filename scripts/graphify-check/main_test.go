package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

// pinnedFingerprint は、テストで既定に使う pin の fingerprint。
const pinnedFingerprint = "d5fd89c46bb5"

// runGit は、dir で git を実行します。失敗は出力ごとテストの失敗にします。
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // 引数はテスト内で組み立てた固定値
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
	)

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

// repositoryWith は、与えたパスにファイルを持つ git リポジトリを作ってそのパスを返します。
// staged は index に載せるパスで、それ以外は未追跡のまま残します。
func repositoryWith(t *testing.T, files map[string]string, staged ...string) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "repo")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	runGit(t, dir, "init", "--quiet")

	for name, body := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	}
	for _, name := range staged {
		runGit(t, dir, "add", name)
	}

	return dir
}

// checkRun は、base 検査を行わない既定条件で run を呼びます。base そのものが主題のケースだけが
// run を直に呼びます。
func checkRun(args []string, list func(string) ([]string, error), read func(string) ([]byte, error)) error {
	return run(args, list, read, func(string, string) ([]string, error) { return nil, nil })
}

// captureLog は標準ロガーの出力を捕まえます。違反の中身は戻り値ではなくログにしか出ないため、
// 「どのファイルがどう汚れているか」の検査はここを通します。
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buffer := &bytes.Buffer{}
	previous := log.Writer()
	flags := log.Flags()
	log.SetOutput(buffer)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(previous)
		log.SetFlags(flags)
	})

	return buffer
}

// staticList は追跡パスを固定で返す列挙手段を作ります。
func staticList(tracked ...string) func(string) ([]string, error) {
	return func(string) ([]string, error) { return tracked, nil }
}

// staticRead は内容を固定で返す読み出し手段を作ります。未登録のパスは読み込み失敗になります。
// pin ファイルは既定で有効なものを返すので、pin 自体が主題のケースだけが上書きします。
func staticRead(bodies map[string]string) func(string) ([]byte, error) {
	return func(name string) ([]byte, error) {
		body, ok := bodies[name]
		if !ok && name == defaultPin {
			return []byte("# comment\ngraphify_version = \"0.9.25\"\nspec_fingerprint = \"" +
				pinnedFingerprint + "\"\n"), nil
		}
		if !ok {
			return nil, os.ErrNotExist
		}

		return []byte(body), nil
	}
}

// captureLog が標準ロガーという共有状態を差し替えるため、Test_run は並列化しない。
// 違反の中身は戻り値ではなくログにしか出ないので、そこを検査する以上ここは直列になる。
//
//nolint:paralleltest // 標準ロガーを捕まえるため並列化できない
func Test_run(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("ホワイトリスト内の成果物だけが追跡され絶対パスも無ければ合格する", func(t *testing.T) {
			buffer := captureLog(t)

			err := checkRun(
				nil,
				staticList("graphify-out/graph.json", "graphify-out/GRAPH_REPORT.md"),
				staticRead(map[string]string{
					"graphify-out/graph.json":      `{"nodes":[{"source_file":"internal/usecase/cart/cart_get_usecase.go"}]}`,
					"graphify-out/GRAPH_REPORT.md": "# Report\n\n/v1/users を辿る\n",
				}),
			)

			require.NoError(t, err)
			assert.Contains(t, buffer.String(), "2 ファイルを検査")
		})

		t.Run("APIパスの小文字usersは絶対パスとして誤検出しない", func(t *testing.T) {
			captureLog(t)

			err := checkRun(
				nil,
				staticList("graphify-out/nodes.json"),
				staticRead(map[string]string{
					"graphify-out/nodes.json": `{"nodes":[{"name":"/v1/users/{userId}"},{"name":"/home"}]}`,
				}),
			)

			require.NoError(t, err)
		})

		t.Run("dirを差し替えると別のディレクトリを検査する", func(t *testing.T) {
			captureLog(t)

			err := checkRun(
				[]string{"-dir", "graphify"},
				staticList("graphify/graph.json"),
				staticRead(map[string]string{"graphify/graph.json": "{}"}),
			)

			require.NoError(t, err)
		})

		t.Run("ヘルプ要求は失敗にしない", func(t *testing.T) {
			captureLog(t)

			err := checkRun([]string{"-h"}, staticList("graphify-out/graph.json"), staticRead(nil))

			require.NoError(t, err)
		})

		t.Run("pinと一致する名前空間のセマンティックキャッシュは追跡してよい", func(t *testing.T) {
			captureLog(t)

			err := checkRun(
				nil,
				staticList(
					"graphify-out/graph.json",
					"graphify-out/cache/semantic/p"+pinnedFingerprint+"/299ab0e9.json",
				),
				staticRead(map[string]string{"graphify-out/graph.json": "{}"}),
			)

			require.NoError(t, err)
		})

		t.Run("specを渡すとpinとの一致を照合して通す", func(t *testing.T) {
			buffer := captureLog(t)
			// promptFingerprint("hello") == 2cf24dba5fb0 なので、その pin と対にする。
			pin := "spec_fingerprint = \"2cf24dba5fb0\"\n"

			err := checkRun(
				[]string{"-spec", "spec.md"},
				staticList("graphify-out/graph.json"),
				staticRead(map[string]string{
					defaultPin:                pin,
					"spec.md":                 "hello",
					"graphify-out/graph.json": "{}",
				}),
			)

			require.NoError(t, err)
			assert.Contains(t, buffer.String(), "抽出プロンプトは pin と一致します")
		})

		t.Run("追跡中の成果物が無ければ合格とは別の文言で対象なしを報告する", func(t *testing.T) {
			buffer := captureLog(t)

			// graphify を回す前の checkout や、出力を丸ごと無視する運用。コミットされる成果物が
			// 無いので落とさないが、「違反が無かった」とは言わない。
			err := checkRun(nil, staticList(), staticRead(nil))

			require.NoError(t, err)
			assert.Contains(t, buffer.String(), "検査対象はありません")
			assert.NotContains(t, buffer.String(), "マシン固有の情報はありません")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("キャッシュが追跡されていれば追跡対象外として落とす", func(t *testing.T) {
			buffer := captureLog(t)

			err := checkRun(
				nil,
				staticList("graphify-out/cache/ast/v0.9.25/3b69f117.json", "graphify-out/graph.json"),
				staticRead(map[string]string{"graphify-out/graph.json": "{}"}),
			)

			require.ErrorIs(t, err, errUnexpectedTracked)
			assert.Contains(t, buffer.String(), "graphify-out/cache/ast/v0.9.25/3b69f117.json")
			assert.Contains(t, buffer.String(), ".gitignore")
		})

		t.Run("マシンローカルなドットファイルが追跡されていれば追跡対象外として落とす", func(t *testing.T) {
			captureLog(t)

			err := checkRun(
				nil,
				staticList("graphify-out/.graphify_python"),
				staticRead(nil),
			)

			require.ErrorIs(t, err, errUnexpectedTracked)
		})

		t.Run("dirの外のパスが混ざっていれば追跡対象外として落とす", func(t *testing.T) {
			captureLog(t)

			// graphify-out/ の接頭辞を持たないため、ファイル名がホワイトリストと一致しても
			// 通してはならない。
			err := checkRun(nil, staticList("graph.json"), staticRead(nil))

			require.ErrorIs(t, err, errUnexpectedTracked)
		})

		t.Run("成果物が絶対パスを含んでいれば混入として落とす", func(t *testing.T) {
			buffer := captureLog(t)

			err := checkRun(
				nil,
				staticList("graphify-out/graph.json"),
				staticRead(map[string]string{
					"graphify-out/graph.json": `{"nodes":[{"id":"users_tomy_dev_src_go_boilerplate_internal_usecase_cart"}],` +
						`"origin_file":"/Users/tomy/dev/src/go-boilerplate/internal/usecase/cart/cart_get_usecase.go"}`,
				}),
			)

			require.ErrorIs(t, err, errMachineLocalPath)
			assert.Contains(t, buffer.String(), `"/Users/"`)
			assert.Contains(t, buffer.String(), "graphify-out/graph.json")
		})

		t.Run("追跡対象外と絶対パス混入が同時にあれば両方の種類を返す", func(t *testing.T) {
			captureLog(t)

			err := checkRun(
				nil,
				staticList("graphify-out/.graphify_root", "graphify-out/graph.json"),
				staticRead(map[string]string{"graphify-out/graph.json": `{"p":"/home/runner/work"}`}),
			)

			require.ErrorIs(t, err, errUnexpectedTracked)
			require.ErrorIs(t, err, errMachineLocalPath)
		})

		t.Run("成果物を読み込めなければ違反ゼロではなくエラーにする", func(t *testing.T) {
			captureLog(t)

			err := checkRun(nil, staticList("graphify-out/graph.json"), staticRead(nil))

			require.ErrorIs(t, err, os.ErrNotExist)
			assert.Contains(t, err.Error(), "graphify-out/graph.json")
		})

		t.Run("追跡パスを列挙できなければ違反ゼロではなくエラーにする", func(t *testing.T) {
			captureLog(t)
			sentinel := xerrors.New("git unavailable")

			err := checkRun(nil, func(string) ([]string, error) { return nil, sentinel }, staticRead(nil))

			require.ErrorIs(t, err, sentinel)
		})

		t.Run("別プロンプトで焼いたキャッシュが追跡されていれば落とす", func(t *testing.T) {
			buffer := captureLog(t)

			// 今回の取り違え（compact 版 spec で焼いた pa567fc138e3a）がそのままコミットされる経路。
			err := checkRun(
				nil,
				staticList("graphify-out/cache/semantic/pa567fc138e3a/299ab0e9.json"),
				staticRead(nil),
			)

			require.ErrorIs(t, err, errSpecPinMismatch)
			assert.Contains(t, buffer.String(), "pa567fc138e3a")
		})

		t.Run("specがpinと食い違えば抽出前に落とす", func(t *testing.T) {
			captureLog(t)

			err := checkRun(
				[]string{"-spec", "spec.md"},
				staticList("graphify-out/graph.json"),
				staticRead(map[string]string{"spec.md": "compact variant"}),
			)

			require.ErrorIs(t, err, errSpecPinMismatch)
			assert.Contains(t, err.Error(), pinnedFingerprint)
		})

		t.Run("specを読み込めなければエラーにする", func(t *testing.T) {
			captureLog(t)

			err := checkRun([]string{"-spec", "absent.md"}, staticList("graphify-out/graph.json"), staticRead(nil))

			require.ErrorIs(t, err, os.ErrNotExist)
			assert.Contains(t, err.Error(), "absent.md")
		})

		t.Run("pinを読み込めなければ検査せずエラーにする", func(t *testing.T) {
			captureLog(t)
			called := false

			err := checkRun(
				[]string{"-pin", "absent.toml"},
				func(string) ([]string, error) { called = true; return nil, nil },
				staticRead(nil),
			)

			require.ErrorIs(t, err, os.ErrNotExist)
			assert.False(t, called)
		})

		t.Run("未知のフラグは失敗にする", func(t *testing.T) {
			captureLog(t)

			err := checkRun([]string{"-unknown"}, staticList("graphify-out/graph.json"), staticRead(nil))

			require.Error(t, err)
		})
	})
}

//nolint:paralleltest // t.Chdir を使うため並列化できない
func Test_listTracked(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("コミット前でもindexに載っていれば昇順で返す", func(t *testing.T) {
			// git ls-files は index を読むので、最初のコミットを作る瞬間にゲートが素通りしない。
			t.Chdir(repositoryWith(t,
				map[string]string{
					"graphify-out/nodes.json":       "{}",
					"graphify-out/graph.json":       "{}",
					"graphify-out/cache/entry.json": "{}",
					"untouched.txt":                 "x",
				},
				"graphify-out",
			))

			tracked, err := listTracked("graphify-out")

			require.NoError(t, err)
			assert.Equal(t, []string{
				"graphify-out/cache/entry.json",
				"graphify-out/graph.json",
				"graphify-out/nodes.json",
			}, tracked)
		})

		t.Run("ディスクにあっても未追跡なら空を返す", func(t *testing.T) {
			t.Chdir(repositoryWith(t, map[string]string{"graphify-out/graph.json": "{}"}))

			tracked, err := listTracked("graphify-out")

			require.NoError(t, err)
			assert.Empty(t, tracked)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("gitリポジトリの外では空ではなくエラーを返す", func(t *testing.T) {
			t.Chdir(t.TempDir())

			tracked, err := listTracked("graphify-out")

			require.Error(t, err)
			assert.Nil(t, tracked)
		})
	})
}

//nolint:paralleltest // 標準ロガーを捕まえるため並列化できない
func Test_verifyBase(t *testing.T) {
	// staticDiff は変更パスを固定で返す差分手段を作ります。
	staticDiff := func(changed ...string) func(string, string) ([]string, error) {
		return func(string, string) ([]string, error) { return changed, nil }
	}

	t.Run("正常系", func(t *testing.T) {
		t.Run("出力に触れていなければ通す", func(t *testing.T) {
			captureLog(t)

			err := run(
				[]string{"-base", "origin/release/v2.1.0"},
				staticList("graphify-out/graph.json"),
				staticRead(map[string]string{"graphify-out/graph.json": "{}"}),
				staticDiff(),
			)

			require.NoError(t, err)
		})

		t.Run("baseを省略すれば差分を取りに行かない", func(t *testing.T) {
			captureLog(t)
			called := false

			err := run(
				nil,
				staticList("graphify-out/graph.json"),
				staticRead(map[string]string{"graphify-out/graph.json": "{}"}),
				func(string, string) ([]string, error) { called = true; return nil, nil },
			)

			require.NoError(t, err)
			assert.False(t, called)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("出力を触っていれば戻し方を添えて落とす", func(t *testing.T) {
			buffer := captureLog(t)

			err := run(
				[]string{"-base", "origin/release/v2.1.0"},
				staticList("graphify-out/graph.json"),
				staticRead(nil),
				staticDiff("graphify-out/graph.json", "graphify-out/edges.json"),
			)

			require.ErrorIs(t, err, errArtifactInDiff)
			assert.Contains(t, buffer.String(), "graphify-out/edges.json")
			assert.Contains(t, err.Error(), "git checkout origin/release/v2.1.0 -- graphify-out")
		})

		t.Run("件数が上限を超えたら省略して残数を示す", func(t *testing.T) {
			buffer := captureLog(t)
			changed := make([]string, 0, maxListedChanges+3)
			for i := range maxListedChanges + 3 {
				changed = append(changed, fmt.Sprintf("graphify-out/cache/semantic/p/%d.json", i))
			}

			err := run(
				[]string{"-base", "main"}, staticList("graphify-out/graph.json"), staticRead(nil), staticDiff(changed...),
			)

			require.ErrorIs(t, err, errArtifactInDiff)
			assert.Contains(t, buffer.String(), "他 3 件")
		})

		t.Run("差分を取れなければ通さずエラーにする", func(t *testing.T) {
			captureLog(t)
			sentinel := xerrors.New("unknown revision")

			err := run(
				[]string{"-base", "nope"},
				staticList("graphify-out/graph.json"),
				staticRead(nil),
				func(string, string) ([]string, error) { return nil, sentinel },
			)

			require.ErrorIs(t, err, sentinel)
		})

		t.Run("base検査は追跡ファイルの検査より先に効く", func(t *testing.T) {
			captureLog(t)
			called := false

			// 出力を持ち込んでいる時点で落とすべきで、その中身の良し悪しは次の問題である。
			err := run(
				[]string{"-base", "main"},
				func(string) ([]string, error) { called = true; return nil, nil },
				staticRead(nil),
				staticDiff("graphify-out/graph.json"),
			)

			require.ErrorIs(t, err, errArtifactInDiff)
			assert.False(t, called)
		})
	})
}

func Test_promptFingerprint(t *testing.T) {
	t.Parallel()

	// 期待値は graphify.cache.prompt_fingerprint に同じ入力を通して採った実測値。ここが
	// upstream とずれると名前空間が食い違い、pin もキャッシュも意味を失う。
	cases := map[string]struct {
		body string
		want string
	}{
		"空文字":             {"", "e3b0c44298fc"},
		"改行なしの1語":         {"hello", "2cf24dba5fb0"},
		"CRLFはLFと同じ結果になる": {"a\r\nb\r\n", "7e18f737311b"},
		"行末の空白は落とす":       {"a   \nb\t\n", "7e18f737311b"},
		"前後の空行と空白は落とす":    {"\n\n  a  \n\n", "ca978112ca1b"},
		"複数行":             {"# spec\n\nFILE_LIST\nCHUNK_PATH\n", "877d2b368e72"},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		for name, tt := range cases {
			t.Run(name+"のfingerprintがgraphify本体と一致する", func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, tt.want, promptFingerprint([]byte(tt.body)))
			})
		}

		t.Run("常に12桁を返す", func(t *testing.T) {
			t.Parallel()

			assert.Len(t, promptFingerprint([]byte("anything")), fingerprintLength)
		})

		t.Run("CRだけの改行もLFへ正規化する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, promptFingerprint([]byte("a\nb")), promptFingerprint([]byte("a\rb")))
		})
	})
}

func Test_readPinnedFingerprint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コメントと空行を読み飛ばしてfingerprintを返す", func(t *testing.T) {
			t.Parallel()
			body := "# 見出し\n\ngraphify_version = \"0.9.25\"\nspec_fingerprint = \"d5fd89c46bb5\"\n"

			got, err := readPinnedFingerprint("pin.toml", staticRead(map[string]string{"pin.toml": body}))

			require.NoError(t, err)
			assert.Equal(t, "d5fd89c46bb5", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読み込めなければエラーにする", func(t *testing.T) {
			t.Parallel()

			_, err := readPinnedFingerprint("absent.toml", staticRead(nil))

			require.ErrorIs(t, err, os.ErrNotExist)
		})

		t.Run("代入として解釈できない行は読み飛ばさずエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "spec_fingerprint = \"d5fd89c46bb5\"\nspec_fingerprint: bare\n"

			_, err := readPinnedFingerprint("pin.toml", staticRead(map[string]string{"pin.toml": body}))

			require.ErrorIs(t, err, errPinInvalidLine)
			assert.Contains(t, err.Error(), "2 行目")
		})

		t.Run("同じキーの再定義は後勝ちにせずエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "spec_fingerprint = \"aaaaaaaaaaaa\"\nspec_fingerprint = \"bbbbbbbbbbbb\"\n"

			_, err := readPinnedFingerprint("pin.toml", staticRead(map[string]string{"pin.toml": body}))

			require.ErrorIs(t, err, errPinDuplicateKey)
		})

		t.Run("spec_fingerprintが無ければエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "graphify_version = \"0.9.25\"\n"

			_, err := readPinnedFingerprint("pin.toml", staticRead(map[string]string{"pin.toml": body}))

			require.ErrorIs(t, err, errPinMissingKey)
		})

		t.Run("空文字のspec_fingerprintは宣言されていないものとして扱う", func(t *testing.T) {
			t.Parallel()
			body := "spec_fingerprint = \"\"\n"

			_, err := readPinnedFingerprint("pin.toml", staticRead(map[string]string{"pin.toml": body}))

			require.ErrorIs(t, err, errPinMissingKey)
		})
	})
}

func Test_checkNamespaces(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pinと一致する名前空間なら違反なし", func(t *testing.T) {
			t.Parallel()

			got := checkNamespaces("graphify-out", pinnedFingerprint, map[string]int{"pd5fd89c46bb5": 818})

			assert.Empty(t, got)
		})

		t.Run("キャッシュが1件も無ければ違反なし", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, checkNamespaces("graphify-out", pinnedFingerprint, map[string]int{}))
		})

		t.Run("同じ名前空間の何百件も1行に畳む", func(t *testing.T) {
			t.Parallel()

			// エントリごとに報告すると同じ文言が数百行並び、肝心の名前空間名が埋もれる。
			got := checkNamespaces("graphify-out", pinnedFingerprint, map[string]int{"pa567fc138e3a": 818})

			require.Len(t, got, 1)
			assert.Contains(t, got[0].detail, "818 件")
			assert.Equal(t, "graphify-out/cache/semantic/pa567fc138e3a", got[0].path)
		})

		t.Run("複数の名前空間は名前順に並べる", func(t *testing.T) {
			t.Parallel()

			got := checkNamespaces("graphify-out", pinnedFingerprint, map[string]int{
				"pbbbbbbbbbbbb": 1, "paaaaaaaaaaaa": 1, "pd5fd89c46bb5": 5,
			})

			require.Len(t, got, 2)
			assert.Contains(t, got[0].path, "paaaaaaaaaaaa")
			assert.Contains(t, got[1].path, "pbbbbbbbbbbbb")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("別プロンプトの名前空間は不一致として落とす", func(t *testing.T) {
			t.Parallel()

			got := checkNamespaces("graphify-out", pinnedFingerprint, map[string]int{"pa567fc138e3a": 2})

			require.Len(t, got, 1)
			assert.Equal(t, errSpecPinMismatch, got[0].kind)
			assert.Contains(t, got[0].detail, "pd5fd89c46bb5")
		})

		t.Run("名前空間を持たないフラット配置は素性不明として落とす", func(t *testing.T) {
			t.Parallel()

			got := checkNamespaces("graphify-out", pinnedFingerprint, map[string]int{"": 3})

			require.Len(t, got, 1)
			assert.Equal(t, errSpecPinMismatch, got[0].kind)
			assert.Contains(t, got[0].detail, "フラット配置")
			assert.Equal(t, "graphify-out/cache/semantic", got[0].path)
		})
	})
}

//nolint:paralleltest // t.Chdir を使うため並列化できない
func Test_diffAgainst(t *testing.T) {
	// committed は 1 コミット済みのリポジトリを作り、その ref 名を返します。
	committed := func(t *testing.T, files map[string]string) string {
		t.Helper()
		dir := repositoryWith(t, files, ".")
		runGit(t, dir, "commit", "--quiet", "-m", "base")

		return dir
	}

	t.Run("正常系", func(t *testing.T) {
		t.Run("出力に変更があればそのパスだけを昇順で返す", func(t *testing.T) {
			dir := committed(t, map[string]string{
				"graphify-out/graph.json": "{}", "graphify-out/nodes.json": "{}", "main.go": "package main",
			})
			t.Chdir(dir)
			require.NoError(t, os.WriteFile("graphify-out/nodes.json", []byte(`{"a":1}`), 0o600))
			require.NoError(t, os.WriteFile("main.go", []byte("package other"), 0o600))

			changed, err := diffAgainst("HEAD", "graphify-out")

			require.NoError(t, err)
			// main.go は dir の外なので含めない。
			assert.Equal(t, []string{"graphify-out/nodes.json"}, changed)
		})

		t.Run("出力に変更が無ければ空を返す", func(t *testing.T) {
			dir := committed(t, map[string]string{"graphify-out/graph.json": "{}", "main.go": "package main"})
			t.Chdir(dir)
			require.NoError(t, os.WriteFile("main.go", []byte("package other"), 0o600))

			changed, err := diffAgainst("HEAD", "graphify-out")

			require.NoError(t, err)
			assert.Empty(t, changed)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("解決できないrefは空ではなくエラーを返す", func(t *testing.T) {
			t.Chdir(committed(t, map[string]string{"graphify-out/graph.json": "{}"}))

			changed, err := diffAgainst("origin/does-not-exist", "graphify-out")

			require.Error(t, err)
			assert.Nil(t, changed)
		})
	})
}

func Test_inspect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ホワイトリストの全ファイルを通す", func(t *testing.T) {
			t.Parallel()
			tracked := make([]string, 0, len(artifacts))
			bodies := make(map[string]string, len(artifacts))
			for _, name := range artifacts {
				path := "graphify-out/" + name
				tracked = append(tracked, path)
				bodies[path] = "clean"
			}

			violations, err := inspect("graphify-out", pinnedFingerprint, tracked, staticRead(bodies))

			require.NoError(t, err)
			assert.Empty(t, violations)
		})

		t.Run("同名でも別ディレクトリなら追跡対象外にする", func(t *testing.T) {
			t.Parallel()

			violations, err := inspect("graphify-out", pinnedFingerprint, []string{"graphify-out/sub/graph.json"}, staticRead(nil))

			require.NoError(t, err)
			require.Len(t, violations, 1)
			assert.Equal(t, errUnexpectedTracked, violations[0].kind)
		})

		t.Run("1ファイルに複数種類の絶対パスがあれば全て報告する", func(t *testing.T) {
			t.Parallel()

			violations, err := inspect(
				"graphify-out",
				pinnedFingerprint,
				[]string{"graphify-out/graph.json"},
				staticRead(map[string]string{"graphify-out/graph.json": `/home/a /Users/b`}),
			)

			require.NoError(t, err)
			assert.Len(t, violations, 2)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読み込み失敗は途中で打ち切ってエラーを返す", func(t *testing.T) {
			t.Parallel()

			violations, err := inspect("graphify-out", pinnedFingerprint, []string{"graphify-out/graph.json"}, staticRead(nil))

			require.ErrorIs(t, err, os.ErrNotExist)
			assert.Nil(t, violations)
		})
	})
}

func Test_scanHomePrefixes(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接頭辞ごとに1件だけを位置の昇順で返す", func(t *testing.T) {
			t.Parallel()
			body := []byte(`/home/one /home/two /Users/three /Users/four`)

			hits := scanHomePrefixes(body)

			require.Len(t, hits, 2)
			assert.Equal(t, "/home/", hits[0].prefix)
			assert.Equal(t, "/Users/", hits[1].prefix)
			assert.Less(t, hits[0].offset, hits[1].offset)
		})

		t.Run("Windowsのホームも検出する", func(t *testing.T) {
			t.Parallel()

			hits := scanHomePrefixes([]byte(`C:\Users\tomy\repo`))

			require.Len(t, hits, 1)
			assert.Equal(t, `C:\Users\`, hits[0].prefix)
		})

		t.Run("rootのホームも検出する", func(t *testing.T) {
			t.Parallel()

			hits := scanHomePrefixes([]byte(`/root/.cache/graphify`))

			require.Len(t, hits, 1)
			assert.Equal(t, "/root/", hits[0].prefix)
		})

		t.Run("小文字のusersは検出しない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, scanHomePrefixes([]byte(`/v1/users/{userId}`)))
		})

		t.Run("該当が無ければ空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, scanHomePrefixes([]byte(`internal/usecase/cart/cart_get_usecase.go`)))
		})

		t.Run("空の内容でも落ちない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, scanHomePrefixes(nil))
		})
	})
}

func Test_excerpt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("前後をexcerptRadiusずつ切り出す", func(t *testing.T) {
			t.Parallel()
			body := []byte(strings.Repeat("a", excerptRadius) + "X" + strings.Repeat("b", excerptRadius))

			got := excerpt(body, excerptRadius)

			assert.Equal(t, strconv.Quote(strings.Repeat("a", excerptRadius)+"X"+strings.Repeat("b", excerptRadius-1)), got)
		})

		t.Run("先頭側はexcerptRadiusに届かなくても0で止まる", func(t *testing.T) {
			t.Parallel()
			body := []byte(strings.Repeat("a", excerptRadius-1) + "X")

			got := excerpt(body, excerptRadius-1)

			assert.Equal(t, strconv.Quote(string(body)), got)
		})

		t.Run("末尾側は内容の長さで止まる", func(t *testing.T) {
			t.Parallel()

			got := excerpt([]byte("X"), 0)

			assert.Equal(t, `"X"`, got)
		})

		t.Run("制御文字は引用符付きで畳まれる", func(t *testing.T) {
			t.Parallel()

			got := excerpt([]byte("a\nb"), 0)

			assert.Equal(t, `"a\nb"`, got)
		})
	})
}
