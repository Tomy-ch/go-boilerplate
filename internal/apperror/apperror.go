// Package apperror は、層の制約無くアプリケーション全体で使用できるエラーを定義します。
package apperror

import "go-boilerplate/pkg/xerrors"

var (
	// ErrInvalidArgument は引数が無効な場合に使用します。
	ErrInvalidArgument = xerrors.New("invalid argument")
	// ErrUnauthenticated は認証に失敗した場合に使用します。
	ErrUnauthenticated = xerrors.New("unauthenticated")
	// ErrPermissionDenied は権限がない場合に使用します。
	ErrPermissionDenied = xerrors.New("permission denied")
	// ErrNotFound は対象が見つからない場合に使用します。
	ErrNotFound = xerrors.New("not found")
	// ErrConflict は競合が発生した場合に使用します。
	ErrConflict = xerrors.New("conflict")
	// ErrValidation は検証が失敗した場合に使用します。
	ErrValidation = xerrors.New("validation error")
	// ErrTooManyRequests はリクエストが多すぎる場合に使用します。
	ErrTooManyRequests = xerrors.New("too many requests")
	// ErrCanceled はクライアントがリクエストをキャンセル/切断した場合に使用します。
	ErrCanceled = xerrors.New("request canceled")
	// ErrInternal はサーバ内部で予期しないエラーが発生した場合に使用します。
	ErrInternal = xerrors.New("internal error")
	// ErrUnimplemented は実装されていない操作が呼び出された場合に使用します。
	ErrUnimplemented = xerrors.New("unimplemented")
	// ErrUnavailable は一時的に利用できない状態を示します（リトライで解消される可能性がある場合）。
	ErrUnavailable = xerrors.New("service unavailable")
)

// appErrors は、定義済みの全 apperror センチネルです。
var appErrors = []error{
	ErrInvalidArgument,
	ErrUnauthenticated,
	ErrPermissionDenied,
	ErrNotFound,
	ErrConflict,
	ErrValidation,
	ErrTooManyRequests,
	ErrCanceled,
	ErrInternal,
	ErrUnimplemented,
	ErrUnavailable,
}

// IsAppError は、err がいずれかの apperror センチネルに該当するかを返します。
func IsAppError(err error) bool {
	for _, sentinel := range appErrors {
		if xerrors.Is(err, sentinel) {
			return true
		}
	}
	return false
}
