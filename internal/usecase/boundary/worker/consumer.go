//go:generate mockgen -source=$GOFILE -destination=mock/mock_consumer.gen.go -package=mock_$GOPACKAGE

package worker

import (
	"context"
	"time"
)

// Consumer は、pull-ack クラスのキューに対する最小 seam です。これ以上広げません。
type Consumer interface {
	// Receive は long-poll で最大 limit 件を取得します。ctx 完了 or メッセージ到着までブロックします。
	Receive(ctx context.Context, limit int) ([]Message, error)
	// Ack は処理成功後にのみ呼びます（process → ack を厳守）。
	Ack(ctx context.Context, m Message) error
	// Nack はメッセージを再配送へ戻します。遅延は保証しません（adapter best-effort）。
	Nack(ctx context.Context, m Message) error
	// Extend は処理 lease/visibility を延長します（長時間 handler のハートビート）。
	Extend(ctx context.Context, m Message, d time.Duration) error
}
