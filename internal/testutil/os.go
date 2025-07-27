package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// ChdirToProjectRoot テスト用にプロジェクトルートに移動する。
//
// 引数に *testing.Mを受け取ることで、この関数の実行タイミングを制御しています。
func ChdirToProjectRoot(m *testing.M) {
	if m == nil {
		panic("ChdirToProjectRoot requires *testing.M argument")
	}

	if os.Getenv("ENV") != "test" {
		panic("ChdirToProjectRoot is only for test use (ENV=test required)")
	}

	dir, err := os.Getwd()
	if err != nil {
		panic("failed to get current working directory: " + err.Error())
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if err = os.Chdir(dir); err != nil {
				panic(
					"failed to change directory to project root: " + err.Error(),
				)
			}
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found in parent dirs")
		}
		dir = parent
	}
}
