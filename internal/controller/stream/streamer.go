package stream

import (
	"context"

	rt "go-boilerplate/internal/usecase/boundary/realtime"

	"github.com/labstack/echo/v5"
)

// StreamRequest は、検証を通った接続の要求です。ticket の束縛と、解決済みの開始位置を運びます。
type StreamRequest struct {
	// Subject は、ticket を発行された subject です。
	Subject string
	// Destination は、接続先を表す stream の識別子です。path の destination と ticket の束縛が一致していることは検証済みです。
	Destination rt.StreamID
	// Scope は、ticket が与えた権限の範囲です（意味は boundary/realtime.StreamGrant.Scope）。
	Scope string
	// Cursor は、この位置より後の event から配信を始めます。replay floor 以上であることは検証済みです。
	Cursor rt.Sequence
	// Revalidate は、接続を索引へ載せた後に ticket をもう一度検証し直します。
	// 検証と登録の間に届いた失効通知は、まだ索引に無いこの接続を取りこぼすためです。nil なら再検証しません。
	Revalidate func(ctx context.Context) error
}

// Streamer は、検証を通った接続に対して SSE のレスポンスを確定し、event を書き続ける境界です。
// 戻るのは接続を閉じるときで、確定後の指示は in-band の control event で client に伝えます。
type Streamer interface {
	// Stream は、req の位置から destination の event を c のレスポンスへ書き続けます。
	Stream(c *echo.Context, req StreamRequest) error
}
