//go:generate oapi-codegen --include-tags=internal/types/error-response --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml

// Package response は、echoでのエラーレスポンスを生成するためのユーティリティを提供します。
package response

import (
	"fmt"
	"net/http"

	"go-boilerplate/internal/controller/error/response/gen"
	"go-boilerplate/pkg/ptr"
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
// err はログ出力用の元エラーとして Internal に格納されます（不要なら nil）。
func NewHTTPErrorFromStatus(httpStatus int, err error, details ...string) *HTTPErrorResponse {
	meta := lookupErrorMetaByHTTPStatus(httpStatus)

	res := newHTTPErrorFromMeta(meta, details...)
	res.Internal = err
	return res
}

// NewInternalErrorResponse は、内部サーバーエラーのエラーレスポンスを生成します。
func NewInternalErrorResponse() *HTTPErrorResponse {
	return NewHTTPErrorFromStatus(http.StatusInternalServerError, nil)
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
