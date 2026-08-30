package realtime

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrInvalidEvent は、DeliveryEvent が必須項目を欠くか JSON として不正な場合のエラーです。
	ErrInvalidEvent = xerrors.Wrap(apperror.ErrValidation, "realtime: invalid delivery event")
	// ErrPayloadTooLarge は、直列化した DeliveryEvent が MaxSerializedBytes を超える場合のエラーです。
	ErrPayloadTooLarge = xerrors.Wrap(apperror.ErrPayloadTooLarge, "realtime: serialized event exceeds the limit")
	// ErrSequenceConflict は、同じ (StreamID, Sequence) に異なる EventID の event が既にある場合のエラーです。
	// 順序の連鎖（ADR-0072）が壊れたことを意味し、retry では解消しません。
	ErrSequenceConflict = xerrors.Wrap(apperror.ErrConflict, "realtime: sequence already holds a different event")
	// ErrReceivingEndGone は、instance の受信先が使えないことを示すエラーです。まだ作っていない場合と、
	// 作った後に外から消された場合（orphan cleanup が停滞した instance を誤って回収した場合）の両方を指します。
	// どちらも受信を再試行しても直らず、受信先を作り直す必要があるため、通常の一時障害と区別して返します。
	// 作り直しは lease を先に書き直す順序が要るので、受信先の実装ではなく順序を持つ側が行います。
	ErrReceivingEndGone = xerrors.Wrap(apperror.ErrUnavailable, "realtime: instance receiving end is gone")
)
