package errorhandler

import (
	"errors"
	"net/http"

	"go-boilerplate/internal/controller/error/response"

	"github.com/getkin/kin-openapi/openapi3filter"
)

// normalizeOpenAPIError は、OpenAPI関連のエラーを正規化し、エラーレスポンスを生成します。
func normalizeOpenAPIError(err error, details ...string) *response.HTTPErrorResponse {
	var (
		reqErr  *openapi3filter.RequestError
		respErr *openapi3filter.ResponseError
		secErr  *openapi3filter.SecurityRequirementsError
	)

	var resErr *response.HTTPErrorResponse

	switch {
	case errors.As(err, &reqErr):
		resErr = response.NewHTTPErrorFromStatus(http.StatusBadRequest, details...)
	case errors.As(err, &secErr):
		resErr = response.NewHTTPErrorFromStatus(http.StatusUnauthorized, details...)
	case errors.As(err, &respErr):
		resErr = response.NewHTTPErrorFromStatus(http.StatusInternalServerError, details...)
	default:
		return nil
	}

	resErr.Internal = err

	return resErr
}
