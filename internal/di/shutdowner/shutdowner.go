//go:generate mockgen -source=$GOFILE -destination=mock/mock_shutdowner.gen.go -package=mock_$GOPACKAGE

// Package shutdowner は、fx.Shutdowner を抽象化し、上位層を fx 依存から切り離すための薄いラッパを提供します。
package shutdowner

import "go.uber.org/fx"

// Shutdowner は、アプリケーションのシャットダウンを要求するインターフェースです。
type Shutdowner interface {
	Shutdown() error
}

type shutdowner struct {
	sd fx.Shutdowner
}

// NewShutdowner は、新しい Shutdowner インスタンスを生成します。
func NewShutdowner(sd fx.Shutdowner) Shutdowner {
	return &shutdowner{sd: sd}
}

func (s *shutdowner) Shutdown() error {
	return s.sd.Shutdown()
}
