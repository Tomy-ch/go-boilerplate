//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package outbox は、トランザクショナル outbox の永続化境界（Store）を定義します。
// emit（usecase 層）と relay engine（controller 層）の双方がこの境界に依存します。
package outbox

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// EmitParams は、業務 tx 内で outbox へ 1 件記録する入力です。
type EmitParams struct {
	// AggregateType は集約種別です（観測・調査用。順序キーではない）。
	AggregateType string
	// AggregateID は集約 ID です（観測・調査用。順序キーではない）。
	AggregateID string
	// EventType はイベント種別 + version です。
	EventType string
	// Payload はイベント本文の JSON（snapshot + version の収束可能なペイロード）です。
	Payload []byte
	// Headers は publish 時に伝搬するヘッダの JSON（traceparent 等）です。空なら '{}' 相当。
	Headers []byte
	// Channel は配送レーンです。既定値を持たないため、必ず指定します。
	Channel Channel
	// OrderingKey は順序保証の単位（ストリーム）です。順序を持たない配送では空にします。
	OrderingKey string
	// OrderingSequence は OrderingKey 内の位置です。OrderingKey が空なら 0 にします。
	OrderingSequence int64
}

// PendingMessage は、claim した未 publish のエントリです。relay engine が publish 対象として扱います。
type PendingMessage struct {
	// ID は outbox エントリの識別子です（mark 系の対象）。
	ID int64
	// MessageID は dedup の安定キーです（Idempotency-Key へ伝搬）。
	MessageID uuid.UUID
	// EventType はイベント種別 + version です。
	EventType string
	// Payload はイベント本文の JSON です。
	Payload []byte
	// Headers は伝搬ヘッダの JSON です。
	Headers []byte
	// Attempts は現時点の publish 試行回数です（診断とバックオフ幅の算出に使い、dead 判定には使いません）。
	Attempts int32
}

// Store は、outbox の永続化境界です。
type Store interface {
	// Insert は、業務 tx 内で outbox へちょうど 1 件記録し、採番された message_id を返します。
	Insert(ctx context.Context, p EmitParams) (uuid.UUID, error)
	// ClaimPending は、指定チャネルの再試行時刻に達した pending エントリを最大 limit 件まで排他取得します。
	// 複数インスタンスが同時に呼び出しても同一のエントリを二重取得しません。
	// 同一 OrderingKey に未 publish の先行エントリがあるエントリは取得しません（stream 内の順序保証）。
	ClaimPending(ctx context.Context, channel Channel, limit int32) ([]PendingMessage, error)
	// MarkPublished は、publish に成功したエントリを published へ遷移します。
	// pending でなければ何もしません（no-op）。
	MarkPublished(ctx context.Context, id int64) error
	// MarkFailed は、直近の失敗理由を記録し、次に claim してよい時刻を nextAttemptAt へ進めます。
	// エントリは pending のまま残り、その時刻まで claim されません。
	MarkFailed(ctx context.Context, id int64, lastErr string, nextAttemptAt time.Time) error
	// MarkDead は、エントリを dead へ遷移します。
	// pending でなければ何もしません（no-op）。
	MarkDead(ctx context.Context, id int64) error
	// ReplayDead は、dead のエントリを pending へ戻します。messageID が nil なら dead のすべてを対象とし、戻した件数を返します。
	ReplayDead(ctx context.Context, messageID *uuid.UUID) (int64, error)
	// DeletePublished は、publish 完了時刻が cutoff より古いエントリを limit 件まで削除し、削除件数を返します（GC）。
	DeletePublished(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
	// OldestPendingCreatedAt は、指定チャネルの最古 pending エントリの作成時刻を返します（SLI=outbox lag 用）。
	// pending のエントリが無ければ ok=false を返します。
	OldestPendingCreatedAt(ctx context.Context, channel Channel) (createdAt time.Time, ok bool, err error)
	// CountBlockedStreams は、先頭エントリが dead であるストリームの数を返します。
	// 先頭が dead のストリームは順序保証により後続が claim されないため、復旧が要る対象です。
	CountBlockedStreams(ctx context.Context, channel Channel) (int64, error)
}
