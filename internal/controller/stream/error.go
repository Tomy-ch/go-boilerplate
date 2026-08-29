package stream

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrCursorMalformed は、client が提示した cursor が形式不正・負数・表現範囲外のときのエラーです。
	ErrCursorMalformed = xerrors.Wrap(apperror.ErrInvalidArgument, "stream cursor is malformed")
	// ErrConnectionCapacity は、この instance の SSE 接続数が上限に達していることを示します。
	// rate limiting ではなく容量の保護で、超過は待たせずに断ります（ADR-0104 / 設計 §2.6）。
	ErrConnectionCapacity = xerrors.Wrap(apperror.ErrUnavailable, "stream connection capacity reached")
	// ErrReplayAdmission は、初回 replay の枠が有界待ちの間に空かなかったことを示します。
	ErrReplayAdmission = xerrors.Wrap(apperror.ErrUnavailable, "initial replay admission timed out")
	// ErrDraining は、instance が停止に入っており新規接続を受け付けないことを示します。
	ErrDraining = xerrors.Wrap(apperror.ErrUnavailable, "instance is draining")
)
