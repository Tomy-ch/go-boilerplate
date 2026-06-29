package auth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrUnauthenticatedSubjectMissing は、JWT 等のトークンに subject クレームが存在しない場合に返す認証エラーです。
	ErrUnauthenticatedSubjectMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: subject missing")
	// ErrSubjectNotUUID は、subject を UUID として解釈できない場合に返す検証エラーです。
	ErrSubjectNotUUID = xerrors.Wrap(apperror.ErrValidation, "id unavailable: subject is not a uuid")
	// ErrTokenMissing は、トークンが指定されていない場合に返すエラーです。
	ErrTokenMissing = xerrors.Wrap(apperror.ErrInvalidArgument, "token missing")
)
