//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import "context"

// EventLogStore は、stream ごとに sequence 順で event を保持する有界の replay store です
// （ADR-0072: 現在状態の正本ではなく、保持期間内の replay のためだけに置く）。
// 失敗は apperror sentinel（読めない・書けないは ErrUnavailable）で返します。
type EventLogStore interface {
	// Append は、event を (StreamID, Sequence) の位置へ 1 度だけ書きます。
	// 同じ位置に同じ EventID の event が既にあれば成功として返します。
	// 同じ位置に異なる EventID の event があれば ErrSequenceConflict を返します。
	// 実装は書く前に event.Validate() を呼び、不正な封筒を保存しません。
	Append(ctx context.Context, event DeliveryEvent) error
	// ReadAfter は、q.After より後ろの event を sequence 昇順に、最大 q.Limit 件返します。
	// 書き込み直後の読み取りでも取りこぼさない（強い一貫性の）読み取りです。
	// 続きがあれば HasMore が true になり、呼び出し側は最後の event の Sequence を次の After に渡します。
	ReadAfter(ctx context.Context, q ReadAfterQuery) (ReadAfterResult, error)
	// Latest は、stream の最後の event を返します。1 件も無ければ ok=false を返します。
	Latest(ctx context.Context, streamID StreamID) (DeliveryEvent, bool, error)
	// Find は、stream の指定 sequence の event を返します。無ければ ok=false を返します。
	Find(ctx context.Context, streamID StreamID, seq Sequence) (DeliveryEvent, bool, error)
	// AppendedThrough は、この stream へ追記した最大の位置を返します。1 度も追記していなければ 0 です
	// （この値の使い方は docs/design/realtime-delivery.md の Glossary「append watermark」）。
	AppendedThrough(ctx context.Context, streamID StreamID) (Sequence, error)
}

// ReadAfterQuery は、cursor 以降の読み取り条件です。
type ReadAfterQuery struct {
	// StreamID は、読む stream です。
	StreamID StreamID
	// After は、この位置より後ろを読みます。0 なら stream の先頭から読みます。
	After Sequence
	// Limit は、1 回で返す最大件数です。0 以下なら実装の既定値になります。
	Limit int32
}

// ReadAfterResult は、1 回分の読み取り結果です。
type ReadAfterResult struct {
	// Events は、sequence 昇順の event です。gap はありません。
	Events []DeliveryEvent
	// HasMore は、Limit（または store の 1 回の読み取り上限）で打ち切られたことを示します。true でも次の読み取りが
	// 0 件のことはあり、false なら続きは無い、という片方向の保証です。
	HasMore bool
}
