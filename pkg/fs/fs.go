// Package fs は、ファイルシステム操作の薄いラッパーを提供します。
// os への直接依存を一点に集約し、利用側はインターフェース経由で差し替え可能にします。
package fs

import (
	"os"
	"path/filepath"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// FS は、ファイルシステム操作を抽象化するインターフェースです。
type FS interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Glob(pattern string) ([]string, error)
}

// OS は、os / path/filepath を用いた FS の実装です。
type OS struct{}

// ReadFile は、name のファイル内容全体を読み込んで返します。
func (OS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // path は呼び出し側で検証された信頼パス
}

// WriteFile は、data を name のファイルへ書き込みます。ファイルが存在しない場合は perm で新規作成し、
// 既に存在する場合はパーミッションを変えずに既存内容を切り詰めてから書き込みます。
func (OS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// Glob は、pattern に一致するファイルパスの一覧を返します。
func (OS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
