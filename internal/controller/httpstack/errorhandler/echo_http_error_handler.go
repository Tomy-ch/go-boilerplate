package errorhandler

import (
	"errors"

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

	// ehe.Internal は nil になり得る（Wrap(nil)=nil で文脈喪失）ため、常に非 nil の err をラップする。
	internal := xerrors.Wrap(err, "echo HTTP error")
	return response.NewHTTPErrorFromStatus(ehe.Code, internal, details...)
}
