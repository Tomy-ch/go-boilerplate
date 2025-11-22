//go:generate oapi-codegen --include-tags=internal/types/error-response --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml

// Package response は、echoでのエラーレスポンスを生成するためのユーティリティを提供します。
package response

import (
	"fmt"
	"net/http"

	"boilerplate-go/internal/controller/error/response/gen"
	"boilerplate-go/pkg/ptr"
)

// HTTPErrorResponse は、HTTPエラーレスポンスの構造体です。
type HTTPErrorResponse struct {
	gen.ErrorResponse
	HTTPStatus int   `json:"-"`
	Internal   error `json:"-"`
}

// NewHTTPErrorFromAppError は、エラーの中身がアプリケーションエラーである場合に、対応するHTTPエラーレスポンスを生成します。
func NewHTTPErrorFromAppError(err error, details ...string) *HTTPErrorResponse {
	meta := lookupErrorMetaByAppError(err)

	res := newHTTPErrorFromMeta(meta, details...)
	res.Internal = err
	return res
}

// NewHTTPErrorFromStatus は、指定されたHTTPステータスコードに対応するHTTPエラーレスポンスを生成します。
func NewHTTPErrorFromStatus(httpStatus int, details ...string) *HTTPErrorResponse {
	meta := lookupErrorMetaByHTTPStatus(httpStatus)

	return newHTTPErrorFromMeta(meta, details...)
}

// NewInternalErrorResponse は、内部サーバーエラーのエラーレスポンスを生成します。
func NewInternalErrorResponse() *HTTPErrorResponse {
	return &HTTPErrorResponse{
		ErrorResponse: gen.ErrorResponse{
			Code:    codeInternalError,
			Message: errorMessageInternalError,
		},
		HTTPStatus: http.StatusInternalServerError,
	}
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
		detailsPtr = ptr.To(details)
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
