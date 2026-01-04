// Package apperror は、層の制約無くアプリケーション全体で使用できるエラーを定義します。
package apperror

import "boilerplate-go/pkg/xerrors"

var (
	// 引数が無効な場合に使用します。400 Bad Request に対応します。
	ErrInvalidArgument = xerrors.New("invalid argument")
	// 認証に失敗した場合に使用します。401 Unauthorised に対応します。
	ErrUnauthenticated = xerrors.New("unauthenticated")
	// 権限がない場合に使用します。403 Forbidden に対応します。
	ErrPermissionDenied = xerrors.New("permission denied")
	// 対象が見つからない場合に使用します。404 Not Found に対応します。
	ErrNotFound = xerrors.New("not found")
	// 競合が発生した場合に使用します。409 Conflict に対応します。
	ErrConflict = xerrors.New("conflict")
	// 検証が失敗した場合に使用します。422 Unprocessable Entity に対応します。
	ErrValidation = xerrors.New("validation error")
	// リクエストが多すぎる場合に使用します。429 Too Many Requests に対応します。
	ErrTooManyRequests = xerrors.New("too many requests")
	// サーバ内部で予期しないエラーが発生した場合に使用します。500 Internal Server Error に対応します。
	ErrInternal = xerrors.New("internal error")
	// 実装されていない操作が呼び出された場合に使用します。501 Not Implemented に対応します。
	ErrUnimplemented = xerrors.New("unimplemented")
	// 一時的に利用できない状態を示します（リトライで解消される可能性がある場合）。503 Service Unavailable に対応します。
	ErrUnavailable = xerrors.New("service unavailable")
)
