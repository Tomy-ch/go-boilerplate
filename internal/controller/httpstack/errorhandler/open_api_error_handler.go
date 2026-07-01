package errorhandler

import (
	"net/http"

	"go-boilerplate/internal/controller/error/response"
	"go-boilerplate/pkg/xerrors"

	"github.com/getkin/kin-openapi/openapi3filter"
)

// normalizeOpenAPIError は、OpenAPI関連のエラーを正規化し、エラーレスポンスを生成します。
func normalizeOpenAPIError(err error, details ...string) *response.HTTPErrorResponse {
	var (
		reqErr  *openapi3filter.RequestError
		secErr  *openapi3filter.SecurityRequirementsError
		respErr *openapi3filter.ResponseError

		resErr *response.HTTPErrorResponse
	)

	switch {
	case xerrors.As(err, &reqErr):
		resErr = response.NewHTTPErrorFromStatus(http.StatusBadRequest, err, details...)
	case xerrors.As(err, &secErr):
		resErr = response.NewHTTPErrorFromStatus(http.StatusUnauthorized, err, details...)
	case xerrors.As(err, &respErr):
		resErr = response.NewHTTPErrorFromStatus(http.StatusInternalServerError, err, details...)
	default:
		return nil
	}

	return resErr
}
