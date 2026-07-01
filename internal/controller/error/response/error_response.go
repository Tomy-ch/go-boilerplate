//go:generate oapi-codegen --include-tags=internal/types/error-response --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml

// Package response は、echoでのエラーレスポンスを生成するためのユーティリティを提供します。
package response

import (
	"fmt"

	"go-boilerplate/internal/controller/error/response/gen"
)

// HTTPErrorResponse は、HTTPエラーレスポンスの構造体です。
//
//nolint:errname // HTTPエラーレスポンスのDTOであり、レスポンス本体を表す名称が適切なため XxxError 形式には改名しない
type HTTPErrorResponse struct {
	gen.ErrorResponse

	HTTPStatus int   `json:"-"`
	Internal   error `json:"-"`
}

// NewHTTPErrorFromAppError は、err をアプリケーションエラーとして解釈し、対応する HTTP エラーレスポンスを返します。
// 既知のエラー型に一致しない場合は 500 Internal Server Error として扱います。
func NewHTTPErrorFromAppError(err error, details ...string) *HTTPErrorResponse {
	meta := lookupErrorMetaByAppError(err)

	res := newHTTPErrorFromMeta(meta, details...)
	res.Internal = err
	return res
}

// NewHTTPErrorFromStatus は、指定されたHTTPステータスコードに対応するHTTPエラーレスポンスを生成します。
// err はログ出力用の元エラーとして Internal に格納されます。
func NewHTTPErrorFromStatus(httpStatus int, err error, details ...string) *HTTPErrorResponse {
	meta := lookupErrorMetaByHTTPStatus(httpStatus)

	res := newHTTPErrorFromMeta(meta, details...)
	res.Internal = err
	return res
}

// Error メソッドは、HTTPエラーレスポンスの文字列表現を返します。
func (e *HTTPErrorResponse) Error() string {
	if e.Internal != nil {
		return e.Internal.Error()
	}
	return fmt.Sprintf("HTTP %d: %s (%s)", e.HTTPStatus, e.Code, e.Message)
}

// newHTTPErrorFromMeta は、指定されたメタ情報からHTTPエラーレスポンスを生成します。
func newHTTPErrorFromMeta(meta httpErrorMeta, details ...string) *HTTPErrorResponse {
	var detailsPtr *[]string
	if len(details) > 0 {
		detailsPtr = new(details)
	}
	return &HTTPErrorResponse{
		ErrorResponse: gen.ErrorResponse{
			Code:    meta.Code,
			Message: meta.Message,
			Details: detailsPtr,
		},
		HTTPStatus: meta.Status,
	}
}
