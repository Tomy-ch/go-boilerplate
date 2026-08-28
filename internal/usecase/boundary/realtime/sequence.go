//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package realtime は、Realtime Delivery の永続化境界を定義します。
package realtime

import (
	"context"
	"strconv"
)

// StreamID は、順序保証の単位となるストリームの識別子です。
type StreamID string

// String は、StreamID の文字列表現を返します。
func (s StreamID) String() string { return string(s) }

// Sequence は、ストリーム内の位置です。1 起算・gap なし・単調増加で、外部表現は 10 進文字列です。
type Sequence int64

// String は、Sequence の外部表現（10 進、ゼロ埋めなし）を返します。
func (s Sequence) String() string { return strconv.FormatInt(int64(s), 10) }

// SequenceAllocator は、ストリームの採番境界です。
type SequenceAllocator interface {
	// Allocate は、業務 tx 内で呼ばれ、ストリームの次の位置を採番します。
	// 採番行のロックは呼び出し側 tx の commit まで保持されるため、同一ストリームの採番は直列化されます。
	Allocate(ctx context.Context, streamID StreamID) (Sequence, error)
	// Current は、ストリームの現在位置を返します。まだ採番されていなければ ok=false を返します。
	Current(ctx context.Context, streamID StreamID) (seq Sequence, ok bool, err error)
}
