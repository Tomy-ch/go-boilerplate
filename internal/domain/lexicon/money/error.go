package money

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// ErrNegativePrice は、単価が負の場合の検証エラーです。
var ErrNegativePrice = xerrors.Wrap(apperror.ErrValidation, "price must not be negative")

// ErrPriceOutOfRange は、単価が決済スケールの整数へ落とせない大きさの場合の検証エラーです。
var ErrPriceOutOfRange = xerrors.Wrap(apperror.ErrValidation, "price exceeds the settlement range")

// ErrInvalidMinorUnit は、最小単位への変換で負の桁数が指定された場合のエラーです（事前条件違反）。
var ErrInvalidMinorUnit = xerrors.Wrap(apperror.ErrValidation, "minor unit digits must not be negative")
