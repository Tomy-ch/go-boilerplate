package realtime

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrInvalidEvent は、DeliveryEvent が必須項目を欠くか JSON として不正な場合のエラーです。
	ErrInvalidEvent = xerrors.Wrap(apperror.ErrValidation, "realtime: invalid delivery event")
	// ErrPayloadTooLarge は、直列化した DeliveryEvent が MaxSerializedBytes を超える場合のエラーです。
	// emit 前に検出され、outbox には書かれません。
	ErrPayloadTooLarge = xerrors.Wrap(apperror.ErrPayloadTooLarge, "realtime: serialized event exceeds the limit")
	// ErrSequenceConflict は、同じ (StreamID, Sequence) に異なる EventID の event が既にある場合のエラーです。
	// 順序の連鎖（ADR-0072）が壊れたことを意味し、retry では解消しません。
	ErrSequenceConflict = xerrors.Wrap(apperror.ErrConflict, "realtime: sequence already holds a different event")
)
