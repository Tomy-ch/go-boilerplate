//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package worker

import "context"

// QueueStatsProvider は、broker queue の滞留量（depth / DLQ count）を取得する任意 capability seam です。
//
// Consumer の必須 interface には含めません。queue depth は broker ごとに意味・取得方法が異なり、
// SQS の値も approximate（強整合でない）ためです。engine はこの capability を知らないまま動作し、
// observability collector だけがこの capability を実装する adapter から滞留量を収集します。
type QueueStatsProvider interface {
	// QueueStats は、source queue と（あれば）DLQ の現在の滞留量を返します。
	QueueStats(ctx context.Context) (QueueStats, error)
}

// QueueStats は、1 worker が対象とする queue 群の滞留量スナップショットです。
type QueueStats struct {
	// Source は、consume 対象の source queue の滞留量です。
	Source QueueDepth
	// DLQ は、DLQ の滞留量です。DLQ を持たない / 取得対象外の場合は nil で、
	// 「0 件」と「取得対象外」を区別します。
	DLQ *QueueDepth
}

// QueueDepth は、queue 内のメッセージ滞留量を状態別に表します。
// broker 非依存の語彙で表現し、SQS 固有の attribute 名は持ち込みません。
type QueueDepth struct {
	// Visible は、受信可能な（可視）メッセージ数です。
	Visible int64
	// InFlight は、受信済みで未確定（処理中・不可視）のメッセージ数です。
	InFlight int64
	// Delayed は、配送遅延中のメッセージ数です。
	Delayed int64
}
