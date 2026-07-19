package auth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrUnauthenticatedSubjectMissing は、JWT 等のトークンに subject クレームが存在しない場合に返す認証エラーです。
	ErrUnauthenticatedSubjectMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: subject missing")
	// ErrUserIDUnresolved は、内部ユーザー ID が未解決の場合に返す認証エラーです。
	ErrUserIDUnresolved = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: user id unresolved")
	// ErrTokenMissing は、トークンが指定されていない場合に返す認証エラーです。
	ErrTokenMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: token missing")
)
