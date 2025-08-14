//go:generate oapi-codegen --include-tags=internal/types/error-response --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml

// Package response は、echoでのエラーレスポンスを生成するためのユーティリティを提供します。
package response

import (
	"fmt"
	"net/http"

	"boilerplate-go/internal/controller/handler/error/response/gen"
	"boilerplate-go/pkg/ptr"
)

// HTTPErrorResponse は、HTTPエラーレスポンスの構造体です。
type HTTPErrorResponse struct {
	gen.ErrorResponse
	HTTPStatus int   `json:"-"`
	Internal   error `json:"-"`
}

// New は、指定されたHTTPステータスコードに基づいてエラーレスポンスを生成します。
func New(
	httpStatus int, err error, details ...string,
) *HTTPErrorResponse {
	meta := lookupErrorMeta(httpStatus)

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
		HTTPStatus: httpStatus,
		Internal:   err,
	}
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
