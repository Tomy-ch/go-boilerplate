package architest

// Realtime Delivery の隔離規約（docs/design/realtime-delivery.md §3.1「Architecture rules (mechanically checked)」）を
// 機械的に検査する。
//
//  1. 機構側の package（boundary/realtime、usecase/realtime、infrastructure の 4 package、controller/stream）は
//     feature の domain / usecase を import しない。
//  2. InstanceLeaseStore を使ってよいのは realtime の package 群、realtime の DI module、orphan cleanup の job / CLI だけ。
//
// import は正規表現で拾う（go/ 配下の package は application code から使わない方針のため）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realtimeMechanismDirs は、規約 1 の対象（機構側の package）。無いディレクトリはまだ実装されていないだけなので飛ばす。
var realtimeMechanismDirs = []string{
	"internal/usecase/boundary/realtime",
	"internal/usecase/realtime",
	"internal/infrastructure/eventlog",
	"internal/infrastructure/streamticket",
	"internal/infrastructure/instancelease",
	"internal/infrastructure/realtimesecret",
	"internal/controller/stream",
}

// instanceLeaseStoreAllowedPrefixes は、規約 2 で InstanceLeaseStore を参照してよいパス（repo root 相対）の prefix。
var instanceLeaseStoreAllowedPrefixes = []string{
	"internal/usecase/boundary/realtime/",
	"internal/usecase/realtime/",
	"internal/infrastructure/instancelease/",
	"internal/di/module/realtime",
	"internal/controller/job/",
	"internal/cli/",
}

// internalImportRe は、import 節の `"go-boilerplate/internal/..."` を捕捉する。
var internalImportRe = regexp.MustCompile(`"(go-boilerplate/internal/[^"]+)"`)

func TestRealtimeDeliveryImportsNoFeature(t *testing.T) {
	t.Parallel()

	violations, err := collectRealtimeFeatureImports(moduleRoot(t), realtimeMechanismDirs)
	require.NoError(t, err)
	assert.Empty(t, violations,
		"Realtime Delivery の機構側 package が feature の domain / usecase を import している（設計正本 §3.1 規約 1）")
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
		"InstanceLeaseStore は realtime の package・realtime の DI module・orphan cleanup の入口からしか参照できない（設計正本 §3.1 規約 2）")
}

// collectRealtimeFeatureImports は、dirs 配下の production code が持つ feature 向けの import を `file -> import` の形で返す。
func collectRealtimeFeatureImports(root string, dirs []string) ([]string, error) {
	var violations []string
	for _, dir := range dirs {
		abs := filepath.Join(root, dir)
		if _, err := os.Stat(abs); err != nil {
			continue
		}

		err := walkProductionGoFiles(abs, func(path string, content []byte) {
			for _, m := range internalImportRe.FindAllStringSubmatch(string(content), -1) {
				if isFeatureImport(m[1]) {
					rel, _ := filepath.Rel(root, path)
					violations = append(violations, filepath.ToSlash(rel)+" -> "+m[1])
				}
			}
		})
		if err != nil {
			return nil, err
		}
	}

	return violations, nil
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
		if strings.Contains(string(content), "InstanceLeaseStore") {
			rel, _ := filepath.Rel(root, path)
			users = append(users, filepath.ToSlash(rel))
		}
	})

	return users, err
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
func walkProductionGoFiles(dir string, fn func(path string, content []byte)) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
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

		content, err := os.ReadFile(path)
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

		t.Run("feature の domain / usecase を import するファイルだけを挙げ、無いディレクトリは飛ばす", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			dir := filepath.Join(root, "internal/usecase/realtime")
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.go"), []byte(
				"package realtime\nimport (\n\t\"go-boilerplate/internal/usecase/boundary/realtime\"\n\t\"go-boilerplate/internal/observability\"\n)\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.go"), []byte(
				"package realtime\nimport \"go-boilerplate/internal/domain/inquiry\"\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "bad_test.go"), []byte(
				"package realtime\nimport \"go-boilerplate/internal/domain/inquiry\"\n"), 0o600))

			got, err := collectRealtimeFeatureImports(root, []string{"internal/usecase/realtime", "internal/controller/stream"})
			require.NoError(t, err)
			assert.Equal(t, []string{"internal/usecase/realtime/bad.go -> go-boilerplate/internal/domain/inquiry"}, got)
		})
	})
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

	root := t.TempDir()
	for rel, body := range map[string]string{
		"internal/a/uses.go":      "package a\nvar _ realtime.InstanceLeaseStore\n",
		"internal/a/uses_test.go": "package a\nvar _ realtime.InstanceLeaseStore\n",
		"internal/a/mock/m.go":    "package mock\nvar _ realtime.InstanceLeaseStore\n",
		"internal/a/gen.gen.go":   "package a\nvar _ realtime.InstanceLeaseStore\n",
		"internal/b/other.go":     "package b\nvar _ int\n",
	} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, rel), []byte(body), 0o600))
	}

	got, err := collectInstanceLeaseStoreUsers(root)
	require.NoError(t, err)
	assert.Equal(t, []string{"internal/a/uses.go"}, got)
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

	root := t.TempDir()
	for rel := range map[string]struct{}{"p/a.go": {}, "p/a_test.go": {}, "p/b.gen.go": {}, "p/mock/c.go": {}, "p/d.txt": {}} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, rel), []byte("x"), 0o600))
	}

	var seen []string
	require.NoError(t, walkProductionGoFiles(root, func(path string, _ []byte) { seen = append(seen, filepath.Base(path)) }))
	assert.Equal(t, []string{"a.go"}, seen)
}
