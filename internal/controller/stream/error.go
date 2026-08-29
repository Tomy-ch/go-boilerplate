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
)

// hintRetryAfter は、レスポンス確定前の拒否に再試行の目安を添えます。確定前の 503 は接続上限・
// 初回 replay の枠・依存の不調の 3 系統あり（設計 §2.3 / §2.6）、client 契約はそのどれにも
// Retry-After を求めます。control event の retryAfterMs と同じ目安で、単位だけが違います
// （ヘッダは秒、control はミリ秒）。
func hintRetryAfter(c *echo.Context) {
	c.Response().Header().Set("Retry-After", strconv.Itoa(int(retryAfterHint.Seconds())))
}
