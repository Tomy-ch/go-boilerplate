package main

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

// lockJSON は lockfileVersion 3 の最小構造を組み立てる。
const lockJSON = `{
  "lockfileVersion": 3,
  "packages": {
    "": { "name": "root", "version": "1.0.0" },
    "node_modules/brace-expansion": { "version": "5.0.7" },
    "node_modules/@stoplight/spectral-core/node_modules/brace-expansion": { "version": "1.1.16" },
    "node_modules/@scope/pkg": { "version": "2.3.4" },
    "node_modules/linked": { "version": "1.0.0", "link": true },
    "node_modules/no-version": {},
    "workspaces/app": { "version": "0.1.0" }
  }
}`

// lockName は監査対象の lockfile 名（root からの相対パス）。
const lockName = "package-lock.json"

var (
	// errRoundTrip は、往復そのものが失敗したことを表すテスト用のエラー。
	errRoundTrip = xerrors.New("round trip failed")
	// errNoWorkDir は、作業ディレクトリを取得できない状況を再現するテスト用のエラー。
	errNoWorkDir = xerrors.New("no working directory")
)

// stubTransport は npm registry への往復を差し替え、パッケージ名ごとの応答を返す。
// 実ネットワークへ出ずに packument の取得経路（URL の組み立て・並列取得・エラー伝播）を検証する。
type stubTransport struct {
	mu       sync.Mutex
	requests []string
	respond  func(name string) (int, string)
}

// unreachableRegistry は、registry へ到達できない状況を再現する RoundTripper。
type unreachableRegistry struct{}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	s.requests = append(s.requests, req.URL.String())
	s.mu.Unlock()

	code, body := s.respond(strings.TrimPrefix(req.URL.Path, "/"))
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func (s *stubTransport) recorded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func writeNpmrc(t *testing.T, dir, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".npmrc"), []byte(body), 0o600))
}

// writeLockfile は name→version から lockfileVersion 3 の lockfile を書き出す。
func writeLockfile(t *testing.T, path string, versions map[string]string) {
	t.Helper()
	packages := map[string]map[string]string{"": {"name": "root"}}
	for name, v := range versions {
		packages[nodeModulesSep+name] = map[string]string{"version": v}
	}
	body, err := json.Marshal(map[string]any{"lockfileVersion": 3, "packages": packages})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, body, 0o600))
}

// packument は version→公開日時だけを持つ最小の packument JSON を返す。
func packument(t *testing.T, times map[string]time.Time) string {
	t.Helper()
	body, err := json.Marshal(struct {
		Time map[string]time.Time `json:"time"`
	}{Time: times})
	require.NoError(t, err)
	return string(body)
}

func (unreachableRegistry) RoundTrip(*http.Request) (*http.Response, error) { return nil, errRoundTrip }

// stubRegistry は npm registry への往復を差し替えたクライアントを返す。
func stubRegistry(respond func(name string) (int, string)) (*http.Client, *stubTransport) {
	stub := &stubTransport{respond: respond}
	return &http.Client{Transport: stub}, stub
}

// offlineClient は、どのリクエストも往復に失敗するクライアントを返す。実ネットワークへ出ないことを
// 保証したい経路の検証にも使う。
func offlineClient() *http.Client { return &http.Client{Transport: unreachableRegistry{}} }

// dirEntry は path の実体から fs.DirEntry を組む。
func dirEntry(t *testing.T, path string) fs.DirEntry {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return fs.FileInfoToDirEntry(info)
}

// useGitStub は、base 側 lockfile を返すダミー git を PATH 先頭へ載せる。
func useGitStub(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" + body
	require.NoError(t, os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755)) //nolint:gosec // 実行可能スタブ
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// captureLog は log の出力先を差し替え、書き込まれた内容を読めるようにする。
func captureLog(t *testing.T) *strings.Builder {
	t.Helper()
	var b strings.Builder
	log.SetOutput(&b)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &b
}

// daysAgo は「N 日前」の時刻を返す（境界で日数が揺れないよう 1 時間ずらす）。
func daysAgo(days int) time.Time {
	return time.Now().UTC().Add(-time.Duration(days)*hoursPerDay*time.Hour - time.Hour)
}

// workDirOf は、常に dir を返す作業ディレクトリの取得手段を組む。
func workDirOf(dir string) func() (string, error) {
	return func() (string, error) { return dir, nil }
}

// auditRoot は、min-release-age を宣言した lockfile 1 本だけを持つ監査対象ディレクトリを作る。
func auditRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeNpmrc(t, root, "min-release-age=7\n")
	writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})
	return root
}

// registryPublishedAt は、どのパッケージにも pkg-a@1.0.0 の公開日時だけを返す registry を組む。
func registryPublishedAt(t *testing.T, at time.Time) *http.Client {
	t.Helper()
	client, _ := stubRegistry(func(string) (int, string) {
		return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": at})
	})
	return client
}

func Test_entry_key(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("name@version を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "@scope/pkg@2.3.4", entry{name: "@scope/pkg", version: "2.3.4"}.key())
		})
	})
}

func Test_parseLock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("node_modules 配下の実パッケージだけを取り出す", func(t *testing.T) {
			t.Parallel()
			got, err := parseLock([]byte(lockJSON))
			require.NoError(t, err)

			keys := make([]string, 0, len(got))
			for _, e := range got {
				keys = append(keys, e.key())
			}
			assert.ElementsMatch(t, []string{
				"brace-expansion@5.0.7",
				"brace-expansion@1.1.16",
				"@scope/pkg@2.3.4",
			}, keys)
		})

		t.Run("ネストしたコピーは最後の node_modules 以降を名前とする", func(t *testing.T) {
			t.Parallel()
			got, err := parseLock([]byte(lockJSON))
			require.NoError(t, err)

			var nested entry
			for _, e := range got {
				if e.version == "1.1.16" {
					nested = e
				}
			}
			assert.Equal(t, "brace-expansion", nested.name)
			assert.Equal(t, "node_modules/@stoplight/spectral-core/node_modules/brace-expansion", nested.path)
		})

		t.Run("スコープ付きパッケージ名を保持する", func(t *testing.T) {
			t.Parallel()
			got, err := parseLock([]byte(lockJSON))
			require.NoError(t, err)

			var scoped entry
			for _, e := range got {
				if e.version == "2.3.4" {
					scoped = e
				}
			}
			assert.Equal(t, "@scope/pkg", scoped.name)
		})

		t.Run("ルート・link・version 無し・workspace は対象外", func(t *testing.T) {
			t.Parallel()
			got, err := parseLock([]byte(lockJSON))
			require.NoError(t, err)

			for _, e := range got {
				assert.NotEqual(t, "linked", e.name)
				assert.NotEqual(t, "no-version", e.name)
				assert.NotEqual(t, "workspaces/app", e.path)
			}
			assert.Len(t, got, 3)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON として壊れていればエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := parseLock([]byte("{"))
			require.Error(t, err)
		})
	})
}

func Test_minReleaseAge(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("min-release-age の宣言値を読む", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")

			got, err := minReleaseAge(root, lockName)
			require.NoError(t, err)
			assert.Equal(t, 7, got)
		})

		t.Run("コメント行と前後の空白を無視する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "# min-release-age=99\n; min-release-age=98\n  min-release-age = 14 \n")

			got, err := minReleaseAge(root, lockName)
			require.NoError(t, err)
			assert.Equal(t, 14, got)
		})

		t.Run("lockfile と同じ階層の .npmrc を見る", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, filepath.Join(root, "docker", "tools"), "min-release-age=3\n")

			got, err := minReleaseAge(root, filepath.Join("docker", "tools", lockName))
			require.NoError(t, err)
			assert.Equal(t, 3, got)
		})

		t.Run(".npmrc が無ければ 0（守るべき cooldown なし）", func(t *testing.T) {
			t.Parallel()
			got, err := minReleaseAge(t.TempDir(), lockName)
			require.NoError(t, err)
			assert.Zero(t, got)
		})

		t.Run("min-release-age の宣言が無ければ 0", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "registry=https://example.test/\n")

			got, err := minReleaseAge(root, lockName)
			require.NoError(t, err)
			assert.Zero(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数値として解釈できない値はエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=seven\n")

			_, err := minReleaseAge(root, lockName)
			require.Error(t, err)
		})

		// 不在と読めない状態を同じ 0 に倒すと、cooldown を宣言した lockfile が対象外へ滑り落ちる。
		t.Run("不在ではなく読めない .npmrc はエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(root, ".npmrc"), 0o750))

			_, err := minReleaseAge(root, lockName)
			require.Error(t, err)
			assert.NotErrorIs(t, err, os.ErrNotExist)
		})
	})
}

func Test_lockfiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("node_modules と vendor 配下を除外して収集する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for _, rel := range []string{
				"docker/tools",
				"mock-auth-server",
				"docker/tools/node_modules/dep",
				"vendor/somepkg",
			} {
				require.NoError(t, os.MkdirAll(filepath.Join(root, rel), 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(root, rel, lockName), []byte("{}"), 0o600))
			}

			got, err := lockfiles(root)
			require.NoError(t, err)
			assert.Equal(t, []string{
				filepath.Join("docker", "tools", lockName),
				filepath.Join("mock-auth-server", lockName),
			}, got)
		})

		t.Run("lockfile が無ければ空を返す", func(t *testing.T) {
			t.Parallel()
			got, err := lockfiles(t.TempDir())
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// 走査できなかったことを「lockfile なし」に倒すと、監査対象ゼロのまま正常終了してしまう。
		t.Run("走査できない起点はエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := lockfiles(filepath.Join(t.TempDir(), "absent"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "walk")
		})
	})
}

func Test_lockPath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lockfile は root からの相対パスにする", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "docker", "tools", lockName)
			writeLockfile(t, path, map[string]string{})

			got, err := lockPath(root, path, dirEntry(t, path))
			require.NoError(t, err)
			assert.Equal(t, filepath.Join("docker", "tools", lockName), got)
		})

		// 依存の実体まで潜ると、監査すべき宣言ではなく取得済みの中身を数え始める。
		t.Run("node_modules は枝ごと落とす", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "node_modules")
			require.NoError(t, os.MkdirAll(path, 0o750))

			_, err := lockPath(root, path, dirEntry(t, path))
			require.ErrorIs(t, err, filepath.SkipDir)
		})

		t.Run("それ以外のディレクトリは潜って対象外として返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "docker")
			require.NoError(t, os.MkdirAll(path, 0o750))

			got, err := lockPath(root, path, dirEntry(t, path))
			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("lockfile 以外のファイルは対象外として返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "package.json")
			require.NoError(t, os.WriteFile(path, []byte("{}"), 0o600))

			got, err := lockPath(root, path, dirEntry(t, path))
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// 絶対パスのまま収集すると、以降の root との結合が存在しない場所を指す。
		t.Run("root から相対化できないパスはエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, lockName)
			writeLockfile(t, path, map[string]string{})

			_, err := lockPath("relative/root", path, dirEntry(t, path))
			require.Error(t, err)
		})
	})
}

func Test_lockEntries(t *testing.T) { //nolint:paralleltest // useGitStub が t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("base 未指定なら lockfile の全エントリを返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			root := t.TempDir()
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0", "pkg-b": "2.0.0"})

			got, err := lockEntries(context.Background(), root, lockName, "")
			require.NoError(t, err)
			assert.Len(t, got, 2)
		})

		t.Run("base 指定なら base 時点に無いエントリだけへ絞る", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			root := t.TempDir()
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0", "pkg-b": "2.0.0"})
			useGitStub(t, "printf '%s' '"+
				`{"packages":{"node_modules/pkg-a":{"version":"1.0.0"}}}`+"'\n")

			got, err := lockEntries(context.Background(), root, lockName, "origin/main")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "pkg-b@2.0.0", got[0].key())
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("lockfile が読めなければエラーを返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			_, err := lockEntries(context.Background(), t.TempDir(), lockName, "")
			require.Error(t, err)
		})

		t.Run("lockfile が JSON として壊れていればエラーを返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, lockName), []byte("{"), 0o600))

			_, err := lockEntries(context.Background(), root, lockName, "")
			require.Error(t, err)
		})
	})
}

func Test_changedOnly(t *testing.T) { //nolint:paralleltest // useGitStub が t.Setenv を使うため並列化不可
	cur := []entry{
		{path: "node_modules/pkg-a", name: "pkg-a", version: "1.0.0"},
		{path: "node_modules/pkg-b", name: "pkg-b", version: "2.0.0"},
	}

	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("base に無い name@version だけを残す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useGitStub(t, "printf '%s' '"+
				`{"packages":{"node_modules/pkg-a":{"version":"1.0.0"}}}`+"'\n")

			got, err := changedOnly(context.Background(), t.TempDir(), lockName, "origin/main", cur)
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "pkg-b@2.0.0", got[0].key())
		})

		t.Run("巻き上げでパスが変わっても版が同じなら対象外にする", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useGitStub(t, "printf '%s' '"+
				`{"packages":{"node_modules/dep/node_modules/pkg-a":{"version":"1.0.0"},`+
				`"node_modules/pkg-b":{"version":"2.0.0"}}}`+"'\n")

			got, err := changedOnly(context.Background(), t.TempDir(), lockName, "origin/main", cur)
			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("base 側に lockfile が無ければ全件を対象にする", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useGitStub(t, "exit 128\n")

			got, err := changedOnly(context.Background(), t.TempDir(), lockName, "origin/main", cur)
			require.NoError(t, err)
			assert.Equal(t, cur, got)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("base 側 lockfile が JSON として壊れていればエラーを返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			useGitStub(t, "printf '%s' '{'\n")

			_, err := changedOnly(context.Background(), t.TempDir(), lockName, "origin/main", cur)
			require.Error(t, err)
		})
	})
}

func Test_packumentTimes(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("packument の time から版ごとの公開日時を返す", func(t *testing.T) {
			t.Parallel()
			published := time.Date(2026, time.July, 23, 11, 39, 25, 0, time.UTC)
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusOK, packument(t, map[string]time.Time{"5.0.8": published})
			})

			got, err := packumentTimes(t.Context(), client, "brace-expansion")
			require.NoError(t, err)
			assert.Equal(t, published, got["5.0.8"].UTC())
		})

		t.Run("スコープ付きの名前をそのまま registry のパスへ載せる", func(t *testing.T) {
			t.Parallel()
			client, stub := stubRegistry(func(string) (int, string) {
				return http.StatusOK, `{"time":{}}`
			})

			_, err := packumentTimes(t.Context(), client, "@scope/pkg")
			require.NoError(t, err)
			assert.Equal(t, []string{registryBase + "@scope/pkg"}, stub.recorded())
		})

		t.Run("404 は判定不能として空を返す", func(t *testing.T) {
			t.Parallel()
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusNotFound, `{"error":"Not found"}`
			})

			got, err := packumentTimes(t.Context(), client, "private-pkg")
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("200/404 以外は判定不能に倒さずエラーを返す", func(t *testing.T) {
			t.Parallel()
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusInternalServerError, ""
			})

			_, err := packumentTimes(t.Context(), client, "brace-expansion")
			require.ErrorIs(t, err, errRegistryStatus)
		})

		t.Run("packument が JSON として壊れていればエラーを返す", func(t *testing.T) {
			t.Parallel()
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusOK, "{"
			})

			_, err := packumentTimes(t.Context(), client, "brace-expansion")
			require.Error(t, err)
			require.NotErrorIs(t, err, errRegistryStatus)
		})

		// URL に載らない名前を黙って通すと、そのパッケージだけ判定不能として静かに監査から落ちる。
		t.Run("URL として組み立てられない名前はエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := packumentTimes(t.Context(), offlineClient(), "brace\x7fexpansion")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "request ")
		})

		// 到達できないことを空の結果に倒すと、registry 障害が「違反なし」に化ける。
		t.Run("registry へ到達できなければエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := packumentTimes(t.Context(), offlineClient(), "brace-expansion")
			require.ErrorIs(t, err, errRoundTrip)
		})
	})
}

func Test_publishTimes(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一パッケージの複数版を 1 回の packument 取得でまかなう", func(t *testing.T) {
			t.Parallel()
			old := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
			recent := time.Date(2026, time.June, 30, 23, 59, 59, 0, time.UTC)
			client, stub := stubRegistry(func(string) (int, string) {
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": old, "2.0.0": recent})
			})

			got, err := publishTimes(t.Context(), client, []entry{
				{name: "pkg-a", version: "1.0.0"},
				{name: "pkg-a", version: "2.0.0"},
			})
			require.NoError(t, err)
			assert.Len(t, stub.recorded(), 1)
			assert.Equal(t, old, got["pkg-a@1.0.0"].UTC())
			assert.Equal(t, recent, got["pkg-a@2.0.0"].UTC())
		})

		t.Run("registry に公開日の無い版は結果へ含めない", func(t *testing.T) {
			t.Parallel()
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(30)})
			})

			got, err := publishTimes(t.Context(), client, []entry{{name: "pkg-a", version: "9.9.9"}})
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得に失敗したパッケージがあれば結果を返さずエラーにする", func(t *testing.T) {
			t.Parallel()
			client, _ := stubRegistry(func(name string) (int, string) {
				if name == "pkg-b" {
					return http.StatusInternalServerError, ""
				}
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(30)})
			})

			got, err := publishTimes(t.Context(), client, []entry{
				{name: "pkg-a", version: "1.0.0"},
				{name: "pkg-b", version: "1.0.0"},
			})
			require.ErrorIs(t, err, errRegistryStatus)
			assert.Nil(t, got)
		})
	})
}

func Test_auditLock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("min-release-age の宣言が無い lockfile は監査せず対象外にする", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})

			findings, audited, err := auditLock(t.Context(), offlineClient(), root, lockName, "")
			require.NoError(t, err)
			assert.Empty(t, findings)
			assert.Zero(t, audited)
		})

		t.Run("窓の内側で公開された版を違反として報告する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(2)})
			})

			findings, audited, err := auditLock(t.Context(), client, root, lockName, "")
			require.NoError(t, err)
			require.Len(t, findings, 1)
			assert.Equal(t, "pkg-a@1.0.0", findings[0].entry.key())
			assert.Equal(t, 2, findings[0].ageDays)
			assert.Equal(t, 7, findings[0].minAge)
			assert.Equal(t, lockName, findings[0].lockfile)
			assert.Equal(t, 1, audited)
		})

		t.Run("窓を越えている版は違反にしない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(30)})
			})

			findings, audited, err := auditLock(t.Context(), client, root, lockName, "")
			require.NoError(t, err)
			assert.Empty(t, findings)
			assert.Equal(t, 1, audited)
		})

		// 境界を含める向きに間違えると、min-release-age を満たした版まで違反として鳴り続ける。
		t.Run("min-release-age ちょうど経過した版は違反にしない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(7)})
			})

			findings, audited, err := auditLock(t.Context(), client, root, lockName, "")
			require.NoError(t, err)
			assert.Empty(t, findings)
			assert.Equal(t, 1, audited)
		})

		// 1 日足りない版を見逃すと、このツールは境界の直前で何も検出しなくなる。
		t.Run("min-release-age に 1 日足りない版は違反にする", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(6)})
			})

			findings, _, err := auditLock(t.Context(), client, root, lockName, "")
			require.Len(t, findings, 1)
			require.NoError(t, err)
			assert.Equal(t, 6, findings[0].ageDays)
		})

		t.Run("registry に公開日が無い版は判定不能として報告しない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusNotFound, ""
			})

			findings, audited, err := auditLock(t.Context(), client, root, lockName, "")
			require.NoError(t, err)
			assert.Empty(t, findings)
			assert.Equal(t, 1, audited)
		})

		t.Run("対象エントリが無ければ registry を引かずに終える", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{})
			client, stub := stubRegistry(func(string) (int, string) {
				return http.StatusOK, `{"time":{}}`
			})

			findings, audited, err := auditLock(t.Context(), client, root, lockName, "")
			require.NoError(t, err)
			assert.Empty(t, findings)
			assert.Zero(t, audited)
			assert.Empty(t, stub.recorded())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("registry の取得に失敗すればエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusInternalServerError, ""
			})

			_, _, err := auditLock(t.Context(), client, root, lockName, "")
			require.ErrorIs(t, err, errRegistryStatus)
		})

		t.Run(".npmrc が壊れていればエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=seven\n")

			_, _, err := auditLock(t.Context(), offlineClient(), root, lockName, "")
			require.Error(t, err)
		})

		// 読めない lockfile を「対象エントリなし」に倒すと、その lockfile だけ黙って監査から落ちる。
		t.Run("cooldown を宣言した lockfile を読めなければエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")

			_, _, err := auditLock(t.Context(), offlineClient(), root, lockName, "")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "read "+lockName)
		})
	})
}

func Test_auditAll(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数 lockfile の違反を集約し監査件数を合算する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, filepath.Join(root, "a"), "min-release-age=7\n")
			writeNpmrc(t, filepath.Join(root, "b"), "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, "a", lockName), map[string]string{"pkg-a": "1.0.0"})
			writeLockfile(t, filepath.Join(root, "b", lockName), map[string]string{"pkg-b": "1.0.0"})
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(2)})
			})

			findings, audited, err := auditAll(t.Context(), client, root,
				[]string{filepath.Join("a", lockName), filepath.Join("b", lockName)}, "")
			require.NoError(t, err)
			assert.Len(t, findings, 2)
			assert.Equal(t, 2, audited)
		})

		t.Run("違反は公開からの経過日数が短い順に並ぶ", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-old": "1.0.0", "pkg-new": "1.0.0"})
			client, _ := stubRegistry(func(name string) (int, string) {
				if name == "pkg-new" {
					return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(1)})
				}
				return http.StatusOK, packument(t, map[string]time.Time{"1.0.0": daysAgo(5)})
			})

			findings, _, err := auditAll(t.Context(), client, root, []string{lockName}, "")
			require.NoError(t, err)
			require.Len(t, findings, 2)
			assert.Equal(t, "pkg-new", findings[0].entry.name)
			assert.Equal(t, "pkg-old", findings[1].entry.name)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("いずれかの lockfile で失敗すれば中断してエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")
			writeLockfile(t, filepath.Join(root, lockName), map[string]string{"pkg-a": "1.0.0"})
			client, _ := stubRegistry(func(string) (int, string) {
				return http.StatusInternalServerError, ""
			})

			findings, audited, err := auditAll(t.Context(), client, root, []string{lockName}, "")
			require.ErrorIs(t, err, errRegistryStatus)
			assert.Nil(t, findings)
			assert.Zero(t, audited)
		})
	})
}

func Test_report(t *testing.T) { //nolint:paralleltest // captureLog が log の出力先を差し替えるため並列化不可
	fs := []finding{{
		lockfile:  "docker/tools/package-lock.json",
		entry:     entry{name: "brace-expansion", version: "5.0.8"},
		published: time.Date(2026, time.July, 23, 11, 39, 25, 0, time.UTC),
		ageDays:   2,
		minAge:    7,
	}}

	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
		t.Run("違反が無ければ監査件数を添えて違反なしと報告する", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			out := captureLog(t)

			report(nil, 504, "", false)

			assert.Contains(t, out.String(), "cooldown 違反なし")
			assert.Contains(t, out.String(), "504")
			assert.Contains(t, out.String(), "全件")
		})

		t.Run("base 指定時はスコープに base を示す", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			out := captureLog(t)

			report(nil, 12, "origin/release/v2.2.0", false)

			assert.Contains(t, out.String(), "origin/release/v2.2.0 からの追加/変更分")
		})

		t.Run("違反はパッケージ・版・経過日数・lockfile を列挙する", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			out := captureLog(t)

			report(fs, 12, "", false)

			assert.Contains(t, out.String(), "cooldown 違反 1 件")
			assert.Contains(t, out.String(), "brace-expansion 5.0.8: 公開 2 日 (< 7) [docker/tools/package-lock.json]")
		})

		t.Run("github 指定時のみ警告アノテーションを出す", func(t *testing.T) { //nolint:paralleltest // log の出力先を共有するため
			plain := captureLog(t)
			report(fs, 12, "", false)
			assert.NotContains(t, plain.String(), "::warning")

			annotated := captureLog(t)
			report(fs, 12, "", true)
			assert.Contains(t, annotated.String(), "::warning file=docker/tools/package-lock.json::")
		})
	})
}

func Test_summary(t *testing.T) {
	t.Parallel()

	fs := []finding{{
		lockfile:  "docker/tools/package-lock.json",
		entry:     entry{name: "brace-expansion", version: "5.0.8"},
		published: time.Date(2026, time.July, 23, 11, 39, 25, 0, time.UTC),
		ageDays:   2,
		minAge:    7,
	}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("違反が無ければ監査件数のみを述べる", func(t *testing.T) {
			t.Parallel()
			got := summary(nil, 504, "")
			assert.Contains(t, got, "504")
			assert.NotContains(t, got, "min-release-age を満たしていません")
		})

		t.Run("違反はパッケージ・版・経過日数・lockfile を列挙する", func(t *testing.T) {
			t.Parallel()
			got := summary(fs, 12, "origin/release/v2.2.0")

			assert.Contains(t, got, "brace-expansion")
			assert.Contains(t, got, "5.0.8")
			assert.Contains(t, got, "公開 2 日 (< 7)")
			assert.Contains(t, got, "docker/tools/package-lock.json")
		})

		t.Run("バイパスが意図的なら根拠を残すよう促す", func(t *testing.T) {
			t.Parallel()
			got := summary(fs, 12, "")
			assert.Contains(t, got, "判断根拠")
		})

		t.Run("base 指定の有無でスコープの表現が変わる", func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, summary(nil, 1, ""), "全エントリ")
			assert.Contains(t, summary(nil, 1, "origin/main"), "追加/変更された")
		})
	})
}

func Test_appendOutput(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("findings と audited を追記する", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out")
			require.NoError(t, appendOutput(path, 3, 504))

			b, err := os.ReadFile(path) //nolint:gosec // テスト内の一時ファイル
			require.NoError(t, err)
			assert.Contains(t, string(b), "findings=3")
			assert.Contains(t, string(b), "audited=504")
		})

		t.Run("既存の内容を消さずに追記する", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out")
			require.NoError(t, os.WriteFile(path, []byte("prior=1\n"), 0o600))
			require.NoError(t, appendOutput(path, 0, 1))

			b, err := os.ReadFile(path) //nolint:gosec // テスト内の一時ファイル
			require.NoError(t, err)
			assert.Contains(t, string(b), "prior=1")
			assert.Contains(t, string(b), "findings=0")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないディレクトリ配下へは追記できずエラーを返す", func(t *testing.T) {
			t.Parallel()
			require.Error(t, appendOutput(filepath.Join(t.TempDir(), "absent", "out"), 1, 1))
		})
	})
}

//nolint:paralleltest // captureLog と t.Setenv がプロセス共有の状態を差し替えるため並列化不可
func Test_run(t *testing.T) {
	// 実行環境の GITHUB_OUTPUT を持ち込まないよう空にする。設定済みのランナー上で走らせると
	// 監査とは無関係なファイルへ追記してしまう。
	t.Setenv("GITHUB_OUTPUT", "")

	//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("lockfile が 1 本も無ければ監査せずに終える", func(t *testing.T) {
			out := captureLog(t)

			err := run([]string{"audit"}, offlineClient(), workDirOf(t.TempDir()))

			require.NoError(t, err)
			assert.Contains(t, out.String(), "package-lock.json が見つかりません")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("窓を越えたエントリだけなら違反なしとして報告する", func(t *testing.T) {
			out := captureLog(t)

			err := run([]string{"audit"}, registryPublishedAt(t, daysAgo(30)), workDirOf(auditRoot(t)))

			require.NoError(t, err)
			assert.Contains(t, out.String(), "cooldown 違反なし")
		})

		// バイパスはテックリード / アーキテクトの判断であり、CI が塞ぐのは誤り。検知しても
		// 終了コードを変えないことがこのツールの性質そのものなので、ここを退行させてはならない。
		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("窓内のエントリを検出しても正常終了する", func(t *testing.T) {
			out := captureLog(t)

			err := run([]string{"audit"}, registryPublishedAt(t, daysAgo(2)), workDirOf(auditRoot(t)))

			require.NoError(t, err)
			assert.Contains(t, out.String(), "cooldown 違反 1 件")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("base 指定で差分の無いエントリを対象から外す", func(t *testing.T) {
			out := captureLog(t)
			root := auditRoot(t)
			useGitStub(t, "cat "+filepath.Join(root, lockName)+"\n")

			err := run([]string{"audit", "--base=origin/main"}, registryPublishedAt(t, daysAgo(2)), workDirOf(root))

			require.NoError(t, err)
			assert.Contains(t, out.String(), "対象エントリなし")
			assert.NotContains(t, out.String(), "cooldown 違反 1 件")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("github 指定のときだけアノテーションを出す", func(t *testing.T) {
			out := captureLog(t)

			err := run([]string{"audit", "--github"}, registryPublishedAt(t, daysAgo(2)), workDirOf(auditRoot(t)))

			require.NoError(t, err)
			assert.Contains(t, out.String(), "::warning file="+lockName+"::pkg-a@1.0.0")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("summary-out 指定でサマリを書き出す", func(t *testing.T) {
			captureLog(t)
			path := filepath.Join(t.TempDir(), "summary.md")

			err := run([]string{"audit", "--summary-out=" + path}, registryPublishedAt(t, daysAgo(2)), workDirOf(auditRoot(t)))

			require.NoError(t, err)
			body, readErr := os.ReadFile(path) //nolint:gosec // テスト内の一時ファイル
			require.NoError(t, readErr)
			assert.Contains(t, string(body), "`pkg-a@1.0.0`")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("GITHUB_OUTPUT があれば件数を追記する", func(t *testing.T) {
			captureLog(t)
			path := filepath.Join(t.TempDir(), "github-output")
			t.Setenv("GITHUB_OUTPUT", path)

			err := run([]string{"audit"}, registryPublishedAt(t, daysAgo(2)), workDirOf(auditRoot(t)))

			require.NoError(t, err)
			body, readErr := os.ReadFile(path) //nolint:gosec // テスト内の一時ファイル
			require.NoError(t, readErr)
			assert.Contains(t, string(body), "findings=1")
			assert.Contains(t, string(body), "audited=1")
		})

		// ヘルプ要求を解析失敗と同じ扱いにすると、`-h` が異常終了に化ける。
		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("ヘルプ要求は失敗にしない", func(t *testing.T) {
			captureLog(t)

			require.NoError(t, run([]string{"audit", "-h"}, offlineClient(), workDirOf(t.TempDir())))
		})
	})

	//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
	t.Run("異常系", func(t *testing.T) {
		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("サブコマンドが無ければ使い方を示してエラーにする", func(t *testing.T) {
			captureLog(t)

			err := run(nil, offlineClient(), workDirOf(t.TempDir()))

			require.ErrorIs(t, err, errUsage)
			assert.Contains(t, err.Error(), "usage: npm-cooldown audit")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("audit 以外のサブコマンドはエラーにする", func(t *testing.T) {
			captureLog(t)

			require.ErrorIs(t, run([]string{"gate"}, offlineClient(), workDirOf(t.TempDir())), errUsage)
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("解釈できないフラグはエラーにする", func(t *testing.T) {
			captureLog(t)

			err := run([]string{"audit", "--no-such-flag"}, offlineClient(), workDirOf(t.TempDir()))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "フラグを解釈できません")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("作業ディレクトリを取得できなければエラーにする", func(t *testing.T) {
			captureLog(t)

			err := run([]string{"audit"}, offlineClient(), func() (string, error) { return "", errNoWorkDir })

			require.ErrorIs(t, err, errNoWorkDir)
		})

		// 走査に失敗したものを 0 件と同じ扱いにすると、監査したのか見ていないのかを区別できなくなる。
		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("走査の起点が存在しなければエラーにする", func(t *testing.T) {
			captureLog(t)

			err := run([]string{"audit"}, offlineClient(), workDirOf(filepath.Join(t.TempDir(), "absent")))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "lockfile の走査")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("registry を引けなければエラーにする", func(t *testing.T) {
			captureLog(t)

			err := run([]string{"audit"}, offlineClient(), workDirOf(auditRoot(t)))

			require.ErrorIs(t, err, errRoundTrip)
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("サマリを書き出せなければエラーにする", func(t *testing.T) {
			captureLog(t)

			err := run([]string{"audit", "--summary-out=" + t.TempDir()}, registryPublishedAt(t, daysAgo(2)), workDirOf(auditRoot(t)))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "write summary")
		})

		//nolint:paralleltest // 親が captureLog / t.Setenv を使うため並列化不可
		t.Run("GITHUB_OUTPUT へ追記できなければエラーにする", func(t *testing.T) {
			captureLog(t)
			t.Setenv("GITHUB_OUTPUT", t.TempDir())

			err := run([]string{"audit"}, registryPublishedAt(t, daysAgo(2)), workDirOf(auditRoot(t)))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "write GITHUB_OUTPUT")
		})
	})
}
