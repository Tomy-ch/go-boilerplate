package worker

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

const (
	// catRetryable は一時障害（Nack で再配送）。
	catRetryable category = iota
	// catPermanent は永久失敗（FailureHandler へ退避して Ack）。
	catPermanent
	// catFatal はプロセス継続不能（engine 停止）。
	catFatal
)

// category は、Handler が返したエラーの分類です。
type category int

// classify は、非 nil の err を重大度順（Fatal > Permanent > Retryable）で分類します。
// いずれの分類センチネルにも該当しない裸の error は、安全側として Retryable 扱いとします。
func classify(err error) category {
	switch {
	case xerrors.Is(err, apperror.ErrFatal):
		return catFatal
	case xerrors.Is(err, apperror.ErrPermanent):
		return catPermanent
	default:
		return catRetryable
	}
}
