//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package shutdowner は、Fxライブラリのインターフェースを提供します。
//
// このパッケージは、uber.org/fx ライブラリをラップし、テストやモックの作成を容易にします。
package shutdowner

import "go.uber.org/fx"

type Shutdowner interface {
	Shutdown(opt ...fx.ShutdownOption) error
}

type shutdowner struct {
	fx.Shutdowner
}

// NewShutdowner は、新しい Shutdowner インスタンスを生成します。
func NewShutdowner(sd fx.Shutdowner) Shutdowner {
	return &shutdowner{Shutdowner: sd}
}
