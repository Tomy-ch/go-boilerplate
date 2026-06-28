//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

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

// Sleeper は、時間経過の待機を抽象化するインターフェースです。
type Sleeper interface {
	// Sleep は、d 経過まで待機します。
	// ctx が先に done になった場合は即座に ctx.Err() を返します。
	Sleep(ctx context.Context, d time.Duration) error
}
