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
	// Attempts は現時点の publish 試行回数です。
	Attempts int32
}

// Store は、outbox の永続化境界です。
type Store interface {
	// Insert は、業務 tx 内で outbox へちょうど 1 件記録し、採番された message_id を返します。
	Insert(ctx context.Context, p EmitParams) (uuid.UUID, error)
	// ClaimPending は、pending のエントリを最大 limit 件まで排他取得します。
	// 複数インスタンスが同時に呼び出しても同一のエントリを二重取得しません。
	ClaimPending(ctx context.Context, limit int32) ([]PendingMessage, error)
	// MarkPublished は、publish に成功したエントリを published へ遷移します。
	// pending でなければ何もしません（no-op）。
	MarkPublished(ctx context.Context, id int64) error
	// MarkFailed は、試行回数を加算し直近の失敗理由を記録し、加算後の試行回数を返します。
	MarkFailed(ctx context.Context, id int64, lastErr string) (attempts int32, err error)
	// MarkDead は、エントリを dead へ遷移します。
	// pending でなければ何もしません（no-op）。
	MarkDead(ctx context.Context, id int64) error
	// ReplayDead は、dead のエントリを pending へ戻します。messageID が nil なら dead のすべてを対象とし、戻した件数を返します。
	ReplayDead(ctx context.Context, messageID *uuid.UUID) (int64, error)
	// DeletePublished は、publish 完了時刻が cutoff より古いエントリを limit 件まで削除し、削除件数を返します（GC）。
	DeletePublished(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
	// OldestPendingCreatedAt は、最古の pending エントリの作成時刻を返します（SLI=outbox lag 用）。
	// pending のエントリが無ければ ok=false を返します。
	OldestPendingCreatedAt(ctx context.Context) (createdAt time.Time, ok bool, err error)
}
