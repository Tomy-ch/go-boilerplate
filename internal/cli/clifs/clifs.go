// Package clifs は、CLI コマンド共通のファイルシステム操作ラッパーを提供します。
// os への直接依存を一点に集約し、利用側はインターフェース経由で差し替え可能にします。
package clifs

import (
	"os"
	"path/filepath"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// FS は、ファイルシステム操作を抽象化するインターフェースです。
type FS interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Glob(pattern string) ([]string, error)
}

// OS は、os / path/filepath を用いた FS の実装です。
type OS struct{}

func (OS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // path は呼び出し側で検証された信頼パス
}

func (OS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (OS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
