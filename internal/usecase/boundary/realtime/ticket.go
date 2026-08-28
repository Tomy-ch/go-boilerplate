//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import (
	"context"
	"time"
)

// TicketHash は、ticket の生値を一方向 hash した保存形です。生値そのものは保存しません（ADR-0074）。
type TicketHash string

// StreamTicket は、subject が destination へ接続してよいことの記録です。
// 生値は client だけが持ち、店側は Hash で照合します。
type StreamTicket struct {
	// Hash は、ticket 生値の hash です。
	Hash TicketHash
	// Subject は、ticket を発行された認証済み subject です。
	Subject string
	// Destination は、接続してよい stream です。cursor はこの stream に対してだけ有効です。
	Destination StreamID
	// Scope は、feature が定めた権限の範囲（読み取り専用の識別子）です。機構は解釈しません。
	Scope string
	// InitialCursor は、cursor を持たずに接続したときの開始位置です。
	InitialCursor Sequence
	// IssuedAt は、発行時刻です。
	IssuedAt time.Time
	// ExpiresAt は、この ticket で新しい接続を始められる期限です。
	ExpiresAt time.Time
}

// StreamGrant は、検証を通った ticket が接続に与える束縛です。生値は含まず、ticket と違い保存もされません。
// InitialCursor は開始位置であって認可の下限ではありません（design/realtime-delivery.md §2.3）。
type StreamGrant struct {
	// Subject は、ticket を発行された subject です。
	Subject string
	// Destination は、接続を許す stream です。
	Destination StreamID
	// Scope は、feature が ticket に与えた権限の範囲です。機構は解釈しません。
	Scope string
	// InitialCursor は、cursor 無しで接続したときの開始位置です。
	InitialCursor Sequence
}

// StreamTicketStore は、発行済み ticket の保存境界です。失敗は apperror sentinel で返します。
type StreamTicketStore interface {
	// Save は、ticket を保存します。同じ Hash への再保存は上書きです。
	Save(ctx context.Context, ticket StreamTicket) error
	// Find は、hash に対応する ticket を返します。無い、または asOf 時点で ExpiresAt を過ぎている
	// ものは ok=false を返します（期限切れの掃除は保存側の都合であり、判定の正本ではありません）。
	Find(ctx context.Context, hash TicketHash, asOf time.Time) (StreamTicket, bool, error)
	// Invalidate は、subject × destination に発行された ticket をすべて無効にします。
	// 該当が無くてもエラーになりません。
	Invalidate(ctx context.Context, subject string, destination StreamID) error
}
