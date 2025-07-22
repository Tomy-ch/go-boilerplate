//go:generate oapi-codegen --include-tags=internal/types/error-response --package=gen --generate=types -o ./gen/types.gen.go $PJ_DIR/openapi/openapi.gen.yaml
package error

import "boilerplate-go/internal/controller/handler/error/gen"

func NewErrorResponse(httpStatus int, details ...string) gen.ErrorResponse {
	meta, ok := errorMeta[httpStatus]
	if !ok {
		meta = HTTPErrorMeta{
			Code:    codeInternalError,
			Message: errorMessageInternalError,
		}
	}

	return gen.ErrorResponse{
		Code:    meta.Code,
		Message: meta.Message,
		Details: details,
	}
}
