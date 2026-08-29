package aws

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// normalize は、SDK の失敗を apperror sentinel へ正規化します。context の取り消しは ErrCanceled、
// それ以外は substrate に届かなかった・拒まれたの区別なく ErrUnavailable です（wakeup は再送できる）。
// 原因は chain に残すので、呼び出し側は sentinel と原因の両方を errors.Is で見られます。
func normalize(err error, op string) error {
	cause := xerrors.Wrap(err, op)
	if xerrors.Is(err, context.Canceled) || xerrors.Is(err, context.DeadlineExceeded) {
		return xerrors.Join(apperror.ErrCanceled, cause)
	}

	return xerrors.Join(apperror.ErrUnavailable, cause)
}
