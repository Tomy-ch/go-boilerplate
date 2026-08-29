// Package realtime は、serve instance の受信先に届く通知（wakeup / 失効）を受け取って接続側へ渡す consumer engine と、
// instance lease の heartbeat loop を提供します。engine 自体は loop と待機制御だけを担い、通知の意味
// （どの接続を起こすか・閉じるか）は sink の実装が持ちます。
package realtime

import (
	"context"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// WakeupSink は、届いた wakeup を接続側へ渡す受け口です。connection registry（`controller/stream`）が実装します。
// 同じ stream の wakeup が重複して届くことは正常であり、実装は冪等でなければなりません（ADR-0073）。
// 呼び出しは engine の loop を止めるので、実装は接続を起こすだけで、読み直し自体は待ちません。
type WakeupSink interface {
	// Wake は、streamID の event が upTo まで増えたことを伝えます。
	Wake(ctx context.Context, streamID rt.StreamID, upTo rt.Sequence)
}

// RevocationSink は、失効通知を接続側へ渡す受け口です。registry が subject で接続を引いて STOP を送ります。
type RevocationSink interface {
	// Revoke は、subject の destination への権利が取り下げられたことを伝えます。
	Revoke(ctx context.Context, subject string, destination rt.StreamID)
}
