package testUtil

import (
	"os"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/env"
)

var Cfg *config.Config

// SetTestEnv はテスト環境で使用される環境変数 "ENV" を "test" に設定します。
//
// 引数に *testing.Mを受け取ることで、この関数の実行タイミングを制御しています。
func SetTestEnv(m *testing.M) {
	if m == nil {
		panic("SetTestEnv requires *testing.M argument")
	}
	if err := os.Setenv("ENV", "test"); err != nil {
		panic("failed to set ENV to 'test': " + err.Error())
	}
}

// RunWithTestSetup はテストのセットアップを行ます。
func RunWithTestSetup(m *testing.M) int {
	if m == nil {
		panic("SetUpTest requires *testing.M argument")
	}

	SetTestEnv(m)
	ChdirToProjectRoot(m)
	err := env.Load()
	if err != nil {
		panic("failed to load environment variables: " + err.Error())
	}

	Cfg, err = config.New()
	if err != nil {
		panic("failed to create test config: " + err.Error())
	}

	return m.Run()
}
