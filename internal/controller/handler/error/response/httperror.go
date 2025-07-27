package errorresponse

import "net/http"

type HTTPErrorMeta struct {
	Code    string
	Message string
}

const (
	// codeBadRequest は、リクエストに不正または不足がある場合に使用されるエラーコードです。
	codeBadRequest = "BAD_REQUEST"
	// codeUnauthorized は、認証されていないアクセスに対して使用されるエラーコードです。
	codeUnauthorized = "UNAUTHORISED"
	// codeAccessDenied は、アクセス権限がない操作を試みた場合に使用されるエラーコードです。
	codeAccessDenied = "ACCESS_DENIED"
	// codeNotFound は、指定されたリソースが存在しない場合に使用されるエラーコードです。
	codeNotFound = "NOT_FOUND"
	// codeResourceConflict は、既に存在するリソースとの競合が発生した場合に使用されるエラーコードです。
	codeResourceConflict = "RESOURCE_CONFLICT"
	// codeInternalError は、サーバー内部で予期しないエラーが発生した場合に使用されるエラーコードです。
	codeInternalError = "INTERNAL_ERROR"
	// codeNotAvailable は、機能が未実装または一時的に利用不可な場合に使用されるエラーコードです。
	codeNotAvailable = "NOT_AVAILABLE"
)

const (
	// errorBadRequest は、リクエストの内容に誤りがあることを示すエラーコードです。
	errorMessageBadRequest = "入力内容に誤りがあります。再度ご確認ください。"
	// errorUnauthorized は、認証が必要な操作に対して認証されていない場合に使用されるエラーコードです。
	errorMessageUnauthorized = "ログインが必要です。ログインして再度お試しください。"
	// errorAccessDenied は、アクセス権限がない操作を試みた場合に使用されるエラーコードです。
	errorMessageAccessDenied = "この操作を行う権限がありません。"
	// errorNotFound は、指定されたリソースが存在しない場合に使用されるエラーコードです。
	errorMessageNotFound = "お探しの情報が見つかりませんでした。"
	// errorResourceConflict は、既に存在するリソースとの競合が発生した場合に使用されるエラーコードです。
	errorMessageResourceConflict = "既に同じ情報が登録されています。"
	// errorInternalError は、サーバー内部で予期しないエラーが発生した場合に使用されるエラーコードです。
	errorMessageInternalError = "サーバーで予期しないエラーが発生しました。時間をおいて再度お試しください。"
	// errorNotAvailable は、機能が未実装または一時的に利用不可な場合に使用されるエラーコードです。
	errorMessageNotAvailable = "現在この機能はご利用いただけません。しばらくしてから再度お試しください。"
)

var errorMeta = map[int]HTTPErrorMeta{
	http.StatusBadRequest: {
		Code:    codeBadRequest,
		Message: errorMessageBadRequest,
	},
	http.StatusUnauthorized: {
		Code:    codeUnauthorized,
		Message: errorMessageUnauthorized,
	},
	http.StatusForbidden: {
		Code:    codeAccessDenied,
		Message: errorMessageAccessDenied,
	},
	http.StatusNotFound: {
		Code:    codeNotFound,
		Message: errorMessageNotFound,
	},
	http.StatusConflict: {
		Code:    codeResourceConflict,
		Message: errorMessageResourceConflict,
	},
	http.StatusInternalServerError: {
		Code:    codeInternalError,
		Message: errorMessageInternalError,
	},
	http.StatusNotImplemented: {
		Code:    codeNotAvailable,
		Message: errorMessageNotAvailable,
	},
	http.StatusServiceUnavailable: {
		Code:    codeNotAvailable,
		Message: errorMessageNotAvailable,
	},
}
