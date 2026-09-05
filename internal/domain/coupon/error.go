package coupon

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid coupon")
	// ErrInvalidID は、クーポン ID の検証に失敗した場合のエラーです。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidUserID は、受給者のユーザー ID の検証に失敗した場合のエラーです。
	ErrInvalidUserID = xerrors.Wrap(errInvalid, "userID failed")
	// ErrInvalidDiscountKind は、値引き種別の検証に失敗した場合のエラーです。
	ErrInvalidDiscountKind = xerrors.Wrap(errInvalid, "discountKind failed")
	// ErrInvalidDiscountValue は、値引きの値の検証に失敗した場合のエラーです。
	ErrInvalidDiscountValue = xerrors.Wrap(errInvalid, "discountValue failed")
	// ErrInvalidScopeKind は、適用範囲種別の検証に失敗した場合のエラーです。
	ErrInvalidScopeKind = xerrors.Wrap(errInvalid, "scopeKind failed")
	// ErrInvalidScopeTarget は、適用範囲が絞る対象の検証に失敗した場合のエラーです。
	ErrInvalidScopeTarget = xerrors.Wrap(errInvalid, "scopeTarget failed")
	// ErrInvalidDiscount は、値引きが未設定の場合のエラーです。
	ErrInvalidDiscount = xerrors.Wrap(errInvalid, "discount is required")
	// ErrInvalidScope は、適用範囲が未設定の場合のエラーです。
	ErrInvalidScope = xerrors.Wrap(errInvalid, "scope is required")
	// ErrInvalidExpiresAt は、有効期限の検証に失敗した場合のエラーです。
	ErrInvalidExpiresAt = xerrors.Wrap(errInvalid, "expiresAt failed")
	// ErrInvalidIssuedAt は、発行日時の検証に失敗した場合のエラーです。
	ErrInvalidIssuedAt = xerrors.Wrap(errInvalid, "issuedAt failed")
)
