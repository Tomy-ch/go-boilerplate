//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package lifecycle は、アプリケーションのライフサイクル管理に関する機能を提供します。
package lifecycle

import (
	"context"

	"go.uber.org/fx"
)

// Registrar は、開始およびシャットダウン処理を登録するためのインターフェースです。
type Registrar interface {
	// RegisterStart は、アプリケーションの開始時に実行される関数を登録します。
	RegisterStart(start func(ctx context.Context) error)
	// RegisterStop は、アプリケーションのシャットダウン時に実行される関数を登録します。
	RegisterStop(stop func(ctx context.Context) error)
}

// lifecycleRegistrar は、fx.Lifecycleを使用して開始およびシャットダウンフックを登録する実装です。
type lifecycleRegistrar struct {
	lc fx.Lifecycle
}

// NewLifecycleRegistrar は、fx.Lifecycleを使用してLifecycleRegistrarを提供します。
func NewLifecycleRegistrar(lc fx.Lifecycle) Registrar {
	return lifecycleRegistrar{lc: lc}
}

func (r lifecycleRegistrar) RegisterStart(fn func(ctx context.Context) error) {
	r.lc.Append(fx.Hook{
		OnStart: fn,
	})
}

// RegisterStop は、fx.Lifecycleにシャットダウンフックを登録します。
func (r lifecycleRegistrar) RegisterStop(fn func(ctx context.Context) error) {
	r.lc.Append(fx.Hook{
		OnStop: fn,
	})
}
