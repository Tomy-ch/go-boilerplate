package stream

import (
	"strconv"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

var (
	// ErrCursorMalformed は、client が提示した cursor が形式不正・負数・表現範囲外のときのエラーです。
	ErrCursorMalformed = xerrors.Wrap(apperror.ErrInvalidArgument, "stream cursor is malformed")
	// ErrConnectionCapacity は、この instance の SSE 接続数が上限に達していることを示します。
	// rate limiting ではなく容量の保護で、超過は待たせずに断ります（ADR-0108 / 設計 §2.6）。
	ErrConnectionCapacity = xerrors.Wrap(apperror.ErrUnavailable, "stream connection capacity reached")
	// ErrReplayAdmission は、初回 replay の枠が有界待ちの間に空かなかったことを示します。
	ErrReplayAdmission = xerrors.Wrap(apperror.ErrUnavailable, "initial replay admission timed out")
	// ErrDraining は、instance が停止に入っており新規接続を受け付けないことを示します。
	ErrDraining = xerrors.Wrap(apperror.ErrUnavailable, "instance is draining")
	// ErrDegraded は、fan-out から通知を受け取れておらず、新規接続に配信を約束できないことを示します。
	// 新規接続の拒否にだけ使い、既存の接続は閉じません（README「Degraded operation」）。
	ErrDegraded = xerrors.Wrap(apperror.ErrUnavailable, "realtime fan-out is degraded")
)

// hintRetryAfter は、レスポンス確定前の 503 拒否に Retry-After（秒）の目安を添えます。
// client 契約は確定前のどの 503 にも Retry-After を求めます（設計 §2.3 / §2.6 / §4.3）。
func hintRetryAfter(c *echo.Context) {
	c.Response().Header().Set("Retry-After", strconv.Itoa(int(retryAfterHint.Seconds())))
}
