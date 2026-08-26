package auth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrUnauthenticatedSubjectMissing は、認証情報に subject が含まれない場合に返す認証エラーです。
	ErrUnauthenticatedSubjectMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: subject missing")
	// ErrUserIDUnresolved は、内部ユーザー ID が未解決の場合に返す認証エラーです。
	ErrUserIDUnresolved = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: user id unresolved")
	// ErrUserIDZero は、ゼロ値の UUID で内部ユーザー ID を解決しようとした場合に返す認証エラーです。
	ErrUserIDZero = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: user id is zero")
	// ErrTokenMissing は、トークンが指定されていない場合に返す認証エラーです。
	ErrTokenMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: token missing")
	// ErrIdentityNotFound は、issuer + subject に対応する内部ユーザーが存在しない場合に返す認証エラーです。
	ErrIdentityNotFound = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: identity not found")
	// ErrUserUnavailable は、解決した内部ユーザーが利用できない状態（削除済み等）の場合に返す認証エラーです。
	ErrUserUnavailable = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: user unavailable")
)
