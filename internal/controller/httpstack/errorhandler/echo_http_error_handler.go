package errorhandler

import (
	"errors"
	"fmt"

	"boilerplate-go/internal/controller/error/response"
	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

// normalizeEchoHTTPError は、EchoのHTTPエラーを正規化し、エラーレスポンスを生成します。
func normalizeEchoHTTPError(err error, details ...string) *response.HTTPErrorResponse {
	var ehe *echo.HTTPError
	if !errors.As(err, &ehe) || !isErrorStatus(ehe.Code) {
		return nil
	}

	resErr := response.NewHTTPErrorFromStatus(ehe.Code)

	var detailsPtr *[]string
	if len(details) > 0 {
		detailsPtr = ptr.To(details)
	}

	resErr.Internal = xerrors.Wrap(ehe.Internal, fmt.Sprintf("echo HTTP error: %v", err))
	resErr.Details = detailsPtr
	return resErr
}
