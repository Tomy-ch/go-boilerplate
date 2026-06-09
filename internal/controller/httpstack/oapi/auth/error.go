package auth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrInvalidAuthDefaultMode は、既定の認証ポリシーが見つからない場合のサーバ内部エラー。
	ErrInvalidAuthDefaultMode = xerrors.Wrap(apperror.ErrInternal, "default auth policy not found")
	// ErrUnauthorizedTokenMissing は、認証トークンが欠如している場合のエラー。
	ErrUnauthorizedTokenMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "authorization token missing")
	// ErrUnauthorizedInvalidToken は、認証トークンが無効な場合のエラー。
	ErrUnauthorizedInvalidToken = xerrors.Wrap(apperror.ErrUnauthenticated, "invalid token")
	// ErrUnauthorizedTokenNotProvided は、認証トークンが提供されていない場合のエラー。
	ErrUnauthorizedTokenNotProvided = xerrors.Wrap(apperror.ErrUnauthenticated, "authorization token not provided")
	// ErrAuthnSlotNotFound は、Authn スロット未装着（ミドルウェア配線ミス）＝サーバ内部異常。
	ErrAuthnSlotNotFound = xerrors.Wrap(apperror.ErrInternal, "authn slot not found in request context")
)
