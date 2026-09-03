//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package publisher は、ドメインイベントの outbound publish 境界（Publisher）と
// publish 先非依存のメッセージ封筒を定義します。relay engine（controller 層）と
// publish adapter（infrastructure 層）の双方が依存する境界です。
package publisher

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// Message は、publish 先非依存のメッセージ封筒です。outbox のエントリから構築され、
// substrate（HTTP 等）へ載せ替える際の中立表現です。net/http 等の型は露出しません。
type Message struct {
	// MessageID は dedup の安定キーです。受信側へ Idempotency-Key として伝搬されます。
	MessageID uuid.UUID
	// EventType はイベント種別 + version です。
	EventType string
	// Payload はイベント本文（snapshot + version の収束可能なペイロード）です。
	Payload []byte
	// Headers は publish 時に伝搬するヘッダ（traceparent 等）です。engine が解釈せず素通しします。
	Headers map[string]string
	// CreatedAt は、業務 tx が outbox へ記録した時刻です。publish 側が遅れを測るのに使います。
	CreatedAt time.Time
}

// Publisher は、メッセージを publish 先へ送る境界です。
type Publisher interface {
	// Publish は、m を publish 先へ送ります。
	// 送信失敗時はエラーを返し、relay の次 poll で再送されます（at-least-once）。
	// 実装は失敗を apperror.ErrPermanent または apperror.ErrRetryable で分類してよく、
	// どちらも運ばないエラーは一時失敗として扱われます（dead 化の基準は ADR-0058）。
	Publish(ctx context.Context, m Message) error
}
