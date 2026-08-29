package architest

// Realtime Delivery の隔離規約（docs/design/realtime-delivery.md §3.1「Architecture rules (mechanically checked)」）を
// 機械的に検査する。
//
//  1. 機構側の package（boundary/realtime、usecase/realtime、infrastructure の 5 package、controller/stream、controller/realtime）は
//     feature の domain / usecase を import しない。
//  2. InstanceLeaseStore を使ってよいのは realtime の package 群、realtime の DI module、orphan cleanup の
//     job / CLI だけ。
//
// import は正規表現で拾う（go/ 配下の package は application code から使わない方針のため）。

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgfs "go-boilerplate/pkg/fs"
)

// realtimeMechanismDirs は、規約 1 の対象（機構側の package）。無いディレクトリはまだ実装されていない
// だけなので飛ばす。
var realtimeMechanismDirs = []string{
	"internal/usecase/boundary/realtime",
	"internal/usecase/realtime",
	"internal/infrastructure/eventlog",
	"internal/infrastructure/streamticket",
	"internal/infrastructure/instancelease",
	"internal/infrastructure/realtimesecret",
	"internal/infrastructure/realtime",
	"internal/controller/stream",
	"internal/controller/realtime",
}

// instanceLeaseStoreAllowedPrefixes は、規約 2 で InstanceLeaseStore を参照してよいパス（repo root 相対）の
// prefix。
var instanceLeaseStoreAllowedPrefixes = []string{
	"internal/usecase/boundary/realtime/",
	"internal/usecase/realtime/",
	"internal/infrastructure/instancelease/",
	"internal/di/module/realtime.go",
	"internal/controller/job/",
	"internal/cli/",
}

// internalImportRe は、import 節の `"go-boilerplate/internal/..."` を捕捉する。
var internalImportRe = regexp.MustCompile(`"(go-boilerplate/internal/[^"]+)"`)

func TestRealtimeDeliveryImportsNoFeature(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("機構側 package は feature の domain / usecase を import しない", func(t *testing.T) {
			t.Parallel()

			violations, err := collectRealtimeFeatureImports(moduleRoot(t), realtimeMechanismDirs)
			require.NoError(t, err)
			assert.Empty(t, violations,
				"Realtime Delivery の機構側 package が feature の domain / usecase を import している（設計正本 §3.1 規約 1）")
		})

		t.Run("宣言した機構側 package はすべて実在する", func(t *testing.T) {
			t.Parallel()

			// 走査は無いディレクトリを黙って飛ばすため、package の移動で規約 1 の検査対象が空になっても気づけない。
			// 一覧の側で実在を主張して、縮退をここで止める。
			for _, dir := range realtimeMechanismDirs {
				matches, err := pkgfs.OS{}.Glob(filepath.Join(moduleRoot(t), dir))
				require.NoError(t, err)
				assert.NotEmpty(t, matches, "%s が無い。realtimeMechanismDirs の宣言を疑う", dir)
			}
		})
	})
}

func TestInstanceLeaseStoreImportAllowlist(t *testing.T) {
	t.Parallel()

	users, err := collectInstanceLeaseStoreUsers(moduleRoot(t))
	require.NoError(t, err)

	var violations []string
	for _, rel := range users {
		if !isInstanceLeaseStoreAllowed(rel) {
			violations = append(violations, rel)
		}
	}

	assert.Empty(t, violations,
		"InstanceLeaseStore は realtime の package・realtime の DI module・orphan cleanup の入口からしか参照できない"+
			"（設計正本 §3.1 規約 2）")
}

// collectRealtimeFeatureImports は、dirs 配下の production code が持つ feature 向けの import を `file -> import` の形で返す。
func collectRealtimeFeatureImports(root string, dirs []string) ([]string, error) {
	var violations []string
	for _, dir := range dirs {
		abs := filepath.Join(root, dir)
		err := walkProductionGoFiles(abs, func(path string, content []byte) {
			rel, _ := filepath.Rel(root, path)
			violations = append(violations, featureImportViolations(filepath.ToSlash(rel), content)...)
		})
		if err != nil {
			return nil, err
		}
	}

	return violations, nil
}

// featureImportViolations は、1 ファイルの内容から feature 向けの import を `rel -> import` の形で返す。
func featureImportViolations(rel string, content []byte) []string {
	var out []string
	for _, m := range internalImportRe.FindAllStringSubmatch(string(content), -1) {
		if isFeatureImport(m[1]) {
			out = append(out, rel+" -> "+m[1])
		}
	}

	return out
}

// isFeatureImport は、feature の domain / usecase を指す import かを判定する。boundary と realtime の usecase は機構側。
func isFeatureImport(imp string) bool {
	switch {
	case strings.HasPrefix(imp, "go-boilerplate/internal/domain/"):
		return true
	case strings.HasPrefix(imp, "go-boilerplate/internal/usecase/boundary/"):
		return false
	case strings.HasPrefix(imp, "go-boilerplate/internal/usecase/realtime"):
		return false
	case strings.HasPrefix(imp, "go-boilerplate/internal/usecase/"):
		return true
	default:
		return false
	}
}

// collectInstanceLeaseStoreUsers は、internal 配下の production code のうち InstanceLeaseStore を参照するファイルを
// repo root 相対で返す。
func collectInstanceLeaseStoreUsers(root string) ([]string, error) {
	var users []string
	err := walkProductionGoFiles(filepath.Join(root, "internal"), func(path string, content []byte) {
		if usesInstanceLeaseStore(content) {
			rel, _ := filepath.Rel(root, path)
			users = append(users, filepath.ToSlash(rel))
		}
	})

	return users, err
}

// usesInstanceLeaseStore は、内容が InstanceLeaseStore を参照しているかを判定する。
func usesInstanceLeaseStore(content []byte) bool {
	return strings.Contains(string(content), "InstanceLeaseStore")
}

// isInstanceLeaseStoreAllowed は、rel（repo root 相対）が InstanceLeaseStore を参照してよい場所かを判定する。
func isInstanceLeaseStoreAllowed(rel string) bool {
	for _, p := range instanceLeaseStoreAllowedPrefixes {
		if strings.HasPrefix(rel, p) {
			return true
		}
	}

	return false
}

// walkProductionGoFiles は、dir 配下の production code（テスト・生成物・mock を除く .go）を fn へ渡す。
// dir が無ければ何もしない（まだ実装されていない package を許すため）。
func walkProductionGoFiles(dir string, fn func(path string, content []byte)) error {
	matches, err := pkgfs.OS{}.Glob(dir)
	if err != nil || len(matches) == 0 {
		return err
	}

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}

		if d.IsDir() {
			if d.Name() == "mock" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}

			return nil
		}

		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".gen.go") {
			return nil
		}

		content, err := pkgfs.OS{}.ReadFile(path)
		if err != nil {
			return err
		}

		fn(path, content)

		return nil
	})
}

func Test_collectRealtimeFeatureImports(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("規約を守る package では空、無いディレクトリは飛ばす", func(t *testing.T) {
			t.Parallel()

			dirs := []string{"internal/usecase/boundary/realtime", "internal/not/here"}
			got, err := collectRealtimeFeatureImports(moduleRoot(t), dirs)
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})
}

func Test_featureImportViolations(t *testing.T) {
	t.Parallel()

	src := []byte("package x\nimport (\n\t\"go-boilerplate/internal/usecase/boundary/realtime\"\n" +
		"\t\"go-boilerplate/internal/domain/inquiry\"\n)\n")
	assert.Equal(t, []string{"internal/usecase/realtime/x.go -> go-boilerplate/internal/domain/inquiry"},
		featureImportViolations("internal/usecase/realtime/x.go", src))
	assert.Empty(t, featureImportViolations("x.go", []byte("package x\nimport \"go-boilerplate/internal/observability\"\n")))
}

func Test_isFeatureImport(t *testing.T) {
	t.Parallel()

	assert.True(t, isFeatureImport("go-boilerplate/internal/domain/inquiry"))
	assert.True(t, isFeatureImport("go-boilerplate/internal/usecase/inquiry"))
	assert.False(t, isFeatureImport("go-boilerplate/internal/usecase/boundary/realtime"))
	assert.False(t, isFeatureImport("go-boilerplate/internal/usecase/boundary/clock"))
	assert.False(t, isFeatureImport("go-boilerplate/internal/usecase/realtime"))
	assert.False(t, isFeatureImport("go-boilerplate/internal/observability"))
}

func Test_collectInstanceLeaseStoreUsers(t *testing.T) {
	t.Parallel()

	got, err := collectInstanceLeaseStoreUsers(moduleRoot(t))
	require.NoError(t, err)
	assert.Contains(t, got, "internal/usecase/boundary/realtime/lease.go")
	for _, rel := range got {
		assert.NotContains(t, rel, "/mock/", "mock は production code ではない")
		assert.False(t, strings.HasSuffix(rel, "_test.go"))
	}
}

func Test_usesInstanceLeaseStore(t *testing.T) {
	t.Parallel()

	assert.True(t, usesInstanceLeaseStore([]byte("var _ realtime.InstanceLeaseStore")))
	assert.False(t, usesInstanceLeaseStore([]byte("var _ realtime.EventLogStore")))
}

func Test_isInstanceLeaseStoreAllowed(t *testing.T) {
	t.Parallel()

	assert.True(t, isInstanceLeaseStoreAllowed("internal/usecase/boundary/realtime/lease.go"))
	assert.True(t, isInstanceLeaseStoreAllowed("internal/di/module/realtime.go"))
	assert.True(t, isInstanceLeaseStoreAllowed("internal/controller/job/orphancleanup/job.go"))
	assert.False(t, isInstanceLeaseStoreAllowed("internal/usecase/inquiry/feed.go"))
	assert.False(t, isInstanceLeaseStoreAllowed("internal/di/module/infrastructure.go"))
}

func Test_walkProductionGoFiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テスト・生成物を除いた .go だけを渡す", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for _, name := range []string{"a.go", "a_test.go", "b.gen.go", "d.txt"} {
				require.NoError(t, pkgfs.OS{}.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
			}

			var seen []string
			require.NoError(t, walkProductionGoFiles(dir, func(path string, _ []byte) { seen = append(seen, filepath.Base(path)) }))
			assert.Equal(t, []string{"a.go"}, seen)
		})

		t.Run("無いディレクトリは何もしない", func(t *testing.T) {
			t.Parallel()

			missing := filepath.Join(t.TempDir(), "missing")
			require.NoError(t, walkProductionGoFiles(missing, func(string, []byte) { t.Fatal("must not be called") }))
		})

		t.Run("mock と node_modules は丸ごと飛ばす", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for _, sub := range []string{"mock", "node_modules", "impl"} {
				require.NoError(t, pkgfs.OS{}.MkdirAll(filepath.Join(dir, sub), 0o750))
				require.NoError(t, pkgfs.OS{}.WriteFile(filepath.Join(dir, sub, "x.go"), []byte("x"), 0o600))
			}

			var seen []string
			require.NoError(t, walkProductionGoFiles(dir, func(path string, _ []byte) {
				seen = append(seen, filepath.Base(filepath.Dir(path)))
			}))
			assert.Equal(t, []string{"impl"}, seen)
		})
	})
}
