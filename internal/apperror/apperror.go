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
	// ErrUnsupportedMediaType はサポートされていない Content-Type / メディア形式の場合に使用します。
	ErrUnsupportedMediaType = xerrors.New("unsupported media type")
	// ErrPayloadTooLarge はリクエストペイロードが許容サイズを超える場合に使用します。
	ErrPayloadTooLarge = xerrors.New("payload too large")
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

// worker のメッセージ処理分類センチネル。
// engine が Handler の返すエラーを分類して挙動を変えるために使用します
// （Retryable=Nack で再配送 / Permanent=FailureHandler へ退避して Ack / Fatal=engine 停止）。
// これらは HTTP エラー taxonomy ではないため、appErrors（IsAppError 判定対象）には含めません。
var (
	// ErrRetryable は一時障害を示します。engine は Nack で再配送します。
	ErrRetryable = xerrors.New("retryable")
	// ErrPermanent は永久失敗を示します。engine は FailureHandler へ退避してから Ack します。
	ErrPermanent = xerrors.New("permanent")
	// ErrFatal はプロセス継続不能を示します。engine は drain して停止します。
	ErrFatal = xerrors.New("fatal")
)

// appErrors は、定義済みの全 apperror センチネルです。
var appErrors = []error{
	ErrInvalidArgument,
	ErrUnauthenticated,
	ErrPermissionDenied,
	ErrNotFound,
	ErrConflict,
	ErrValidation,
	ErrUnsupportedMediaType,
	ErrPayloadTooLarge,
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
