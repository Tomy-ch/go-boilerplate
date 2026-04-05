package errorhandler

import (
	"errors"
	"fmt"

	"go-boilerplate/internal/controller/error/response"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

// normalizeEchoHTTPError は、EchoのHTTPエラーを正規化し、エラーレスポンスを生成します。
func normalizeEchoHTTPError(err error, details ...string) *response.HTTPErrorResponse {
	var ehe *echo.HTTPError
	if !errors.As(err, &ehe) || !isErrorStatus(ehe.Code) {
		return nil
	}

	resErr := response.NewHTTPErrorFromStatus(ehe.Code, details...)
	resErr.Internal = xerrors.Wrap(ehe.Internal, fmt.Sprintf("echo HTTP error: %v", err))

	return resErr
}
