//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import (
	"context"
)

// SequenceAllocator は、ストリームの採番境界です。
type SequenceAllocator interface {
	// Allocate は、業務 tx 内で呼ばれ、ストリームの次の位置を採番します。
	// 採番行のロックは呼び出し側 tx の commit まで保持されるため、同一ストリームの採番は直列化されます。
	Allocate(ctx context.Context, streamID StreamID) (Sequence, error)
	// Current は、ストリームの現在位置を返します。まだ採番されていなければ ok=false を返します。
	Current(ctx context.Context, streamID StreamID) (seq Sequence, ok bool, err error)
}

// String は、StreamID の文字列表現を返します。
func (s StreamID) String() string { return string(s) }
