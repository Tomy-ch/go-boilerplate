//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package clock は、現在時刻の取得を抽象化するバウンダリインターフェースを提供します。
package clock

import "time"

// Clock は、現在の時刻を取得するためのインターフェースです。
type Clock interface {
	// Now は、現在の時刻を返します。
	Now() time.Time
}
