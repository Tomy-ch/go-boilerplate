//go:generate mockgen -source=$GOFILE -destination=mock/mock_tx_manager.gen.go -package=mock_$GOPACKAGE

// Package tx は、トランザクションの管理を提供します。
package tx

import (
	"context"
)

// Manager は、トランザクションの管理を行うインターフェースです。
//
// Do は serialization failure / deadlock 検出時に fn を最大 N 回再実行しうります。
// よって fn は DB 副作用以外について冪等であること（呼出側責務）。外部副作用（イベント発行・
// メール送信等）は、同一 tx 内で outbox 行として書き込めば rollback と共に巻き戻り retry-safe に
// なります。nested（既存 tx を再利用する）経路はリトライ対象外で 1 回のみ実行されます。
type Manager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// DoWithResult は、トランザクション内で値を取得するためのヘルパー関数です。
func DoWithResult[T any](
	ctx context.Context, m Manager, fn func(ctx context.Context) (T, error),
) (T, error) {
	var result T
	if err := m.Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	}); err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}
