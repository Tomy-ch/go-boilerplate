//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package realtime は、serve instance の受信先に届く通知（wakeup / 失効）を受け取って接続側へ渡す consumer engine と、
// instance lease の heartbeat loop を提供します。engine 自体は loop と待機制御だけを担い、通知の意味
// （どの接続を起こすか・閉じるか）は sink の実装が持ちます。
package realtime

import (
	"context"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// Waker は、届いた wakeup を接続側へ渡す受け口です。
// 同じ stream の wakeup が重複して届くことは正常であり、実装は冪等でなければなりません（ADR-0073）。
// 呼び出しは engine の loop を止めるので、実装は接続を起こすだけで、読み直し自体は待ちません。
type Waker interface {
	// Wake は、streamID の event が upTo まで増えたことを伝えます。
	Wake(ctx context.Context, streamID rt.StreamID, upTo rt.Sequence)
}

// Revoker は、失効通知を接続側へ渡す受け口です。実装は subject の接続を STOP で閉じます。
type Revoker interface {
	// Revoke は、subject の destination への権利が取り下げられたことを伝えます。
	Revoke(ctx context.Context, subject string, destination rt.StreamID)
}
