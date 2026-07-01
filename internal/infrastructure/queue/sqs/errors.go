package sqs

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// normalizeError は、SQS/AWS のエラーを apperror センチネルへ正規化します。
// broker 由来のエラーは一時的（リトライ可能）とみなし ErrUnavailable に寄せます。
func normalizeError(err error) error {
	if err == nil {
		return nil
	}
	if xerrors.Is(err, context.Canceled) {
		return xerrors.Join(apperror.ErrCanceled, err)
	}
	return xerrors.Join(apperror.ErrUnavailable, err)
}
