package driver

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// WithTx は、外部テストパッケージ（driver_test）から未公開の withTx を利用するためのテスト用公開ブリッジです。
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return withTx(ctx, tx)
}
