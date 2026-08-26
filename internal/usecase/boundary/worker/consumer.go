//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package worker

import (
	"context"
	"time"
)

// Consumer は、pull-ack クラスのキューに対する seam です。
type Consumer interface {
	// Receive は long-poll で最大 limit 件を取得します。ctx 完了 or メッセージ到着までブロックします。
	Receive(ctx context.Context, limit int) ([]Message, error)
	// Ack は処理成功後にのみ呼びます（process → ack を厳守）。
	Ack(ctx context.Context, m Message) error
	// Nack はメッセージを即時に再配送へ戻します（遅延なし）。
	Nack(ctx context.Context, m Message) error
	// NackWithBackoff は、最低 d だけ遅延させてからメッセージを再配送へ戻します。
	// adapter は native 機構（SQS ChangeMessageVisibility 等）で遅延を honor します。
	// d<=0 は Nack と等価です。delay は best-effort ではなく port が要求する capability です。
	NackWithBackoff(ctx context.Context, m Message, d time.Duration) error
	// Extend は処理 lease/visibility を延長します（長時間 handler のハートビート）。
	Extend(ctx context.Context, m Message, d time.Duration) error
}
