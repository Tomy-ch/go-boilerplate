package auth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrInvalidAuthDefaultMode は、既定の認証ポリシーが見つからない場合のサーバ内部エラー（予約: 現在は返らない）。
	ErrInvalidAuthDefaultMode = xerrors.Wrap(apperror.ErrInternal, "default auth policy not found")
	// ErrUnauthorizedTokenMissing は、認証トークンが欠如している場合のエラー（予約: 現在は返らない。実際の未提示は ErrUnauthorizedTokenNotProvided）。
	ErrUnauthorizedTokenMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "authorization token missing")
	// ErrUnauthorizedInvalidToken は、認証トークンが無効な場合のエラー。
	ErrUnauthorizedInvalidToken = xerrors.Wrap(apperror.ErrUnauthenticated, "invalid token")
	// ErrUnauthorizedTokenNotProvided は、認証トークンが提供されていない場合のエラー。
	ErrUnauthorizedTokenNotProvided = xerrors.Wrap(apperror.ErrUnauthenticated, "authorization token not provided")
	// ErrUnauthorizedSchemeUnsupported は、operation が宣言した securityScheme を検証できる認証器が配線されていない場合のエラー。
	// 検証できない資格情報は受け入れない（fail-closed）。
	ErrUnauthorizedSchemeUnsupported = xerrors.Wrap(apperror.ErrUnauthenticated, "security scheme is not supported")
	// ErrDuplicateScheme は、同じ securityScheme を担当する認証器が 2 つ以上配線された場合のエラー（結線の不具合）。
	// 後勝ちで黙って片方を捨てると検証が片方にしか効かないため、起動時に落とす。
	ErrDuplicateScheme = xerrors.Wrap(apperror.ErrInternal, "duplicate security scheme authenticator")
	// ErrAuthnSlotNotFound は、Authn スロット未装着の場合のエラー（資格情報と無関係な結線の不具合）。
	ErrAuthnSlotNotFound = xerrors.Wrap(apperror.ErrInternal, "authn slot not found in request context")
)
