package aws

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// normalize は、SDK の失敗を apperror sentinel へ正規化します。context の取り消しは ErrCanceled、
// それ以外は substrate に届かなかった・拒まれたの区別なく ErrUnavailable です（wakeup は再送できる）。
func normalize(err error, op string) error {
	if xerrors.Is(err, context.Canceled) || xerrors.Is(err, context.DeadlineExceeded) {
		return xerrors.Wrap(apperror.ErrCanceled, op+": "+err.Error())
	}

	return xerrors.Wrap(apperror.ErrUnavailable, op+": "+err.Error())
}
