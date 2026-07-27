package errorhandler

import (
	"go-boilerplate/internal/controller/error/response"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

// normalizeEchoHTTPError は、EchoのHTTPエラーを正規化し、エラーレスポンスを生成します。
// ステータスは [echo.StatusCode] で解決するため、[echo.HTTPError] だけでなく
// echo.ErrNotFound のようにステータスだけを持つ定義済みエラーも対象になります。
func normalizeEchoHTTPError(err error, details ...string) *response.HTTPErrorResponse {
	status := echo.StatusCode(err)
	if !isErrorStatus(status) {
		return nil
	}

	// 内包するエラーは nil になり得る（Wrap(nil)=nil で文脈喪失）ため、常に非 nil の err をラップする。
	internal := xerrors.Wrap(err, "echo HTTP error")
	return response.NewHTTPErrorFromStatus(status, internal, details...)
}
