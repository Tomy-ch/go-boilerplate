//go:generate mockgen -source=$GOFILE -destination=mock/mock_clock.gen.go -package=mock_$GOPACKAGE

// Package clock は、現在時刻の取得を抽象化するバウンダリインターフェースを提供します。
package clock

import (
	"context"
	"time"
)

// Clock は、現在の時刻を取得するためのインターフェースです。
type Clock interface {
	// Now は、現在の時刻を返します。
	Now() time.Time
}

// Sleeper は、決定的にテスト可能な待機を提供するインターフェースです。
// backoff のスリープや breaker のタイマー待ちが依存します。
// Now() の消費者を巻き込まないよう Clock とは別インターフェースに分離しています。
type Sleeper interface {
	// Sleep は、d 経過まで待機します。
	// ctx が先に done になった場合は即座に ctx.Err() を返します。
	Sleep(ctx context.Context, d time.Duration) error
}
