//go:generate mockgen -source=$GOFILE -destination=mock/mock_connection.gen.go -package=mock_$GOPACKAGE
package driver

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// txKey は、実行中のトランザクション（pgx.Tx）を context に保持・識別するためのキーです。
type txKey struct{}

// DBTX は、トランザクション・コネクションプールの双方に対して SQL を実行できる最小インターフェースです。
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// New は、context にトランザクションが在ればそれを、無ければ接続プールを DBTX として返します（新規接続は生成しません）。
func New(ctx context.Context, db DatabaseDriver) DBTX {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	if ok {
		return tx
	}
	return db
}
