// Package apperror は、層の制約無くアプリケーション全体で使用できるエラーを定義します。
package apperror

import "boilerplate-go/pkg/xerrors"

var (
	// 引数が無効な場合に使用します。
	ErrInvalidArgument = xerrors.New("invalid argument")
	// 認証に失敗した場合に使用します。
	ErrUnauthenticated = xerrors.New("unauthenticated")
	// 権限がない場合に使用します。
	ErrPermissionDenied = xerrors.New("permission denied")
	// 対象が見つからない場合に使用します。
	ErrNotFound = xerrors.New("not found")
	// 競合が発生した場合に使用します。
	ErrConflict = xerrors.New("conflict")
	// 検証が失敗した場合に使用します。
	ErrValidation = xerrors.New("validation error")
	// サーバ内部で予期しないエラーが発生した場合に使用します。
	ErrInternal = xerrors.New("internal error")
	// 実装されていない操作が呼び出された場合に使用します。
	ErrUnimplemented = xerrors.New("unimplemented")
	// 一時的に利用できない状態を示します（リトライで解消される可能性がある場合）。
	ErrUnavailable = xerrors.New("service unavailable")
)
