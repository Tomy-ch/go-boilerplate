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

	// nil になり得る ehe.Internal ではなく、常に非 nil の err をラップして文脈を保持する
	// （ehe.Internal が nil のとき Wrap(nil) は nil を返し、診断情報が失われるため）。
	internal := xerrors.Wrap(err, "echo HTTP error")
	return response.NewHTTPErrorFromStatus(ehe.Code, internal, details...)
}
