package errorhandler

import (
	"net/http"

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

// setAllowHeader は、405 レスポンスへ許可メソッド一覧を Allow ヘッダーとして付与します。
// status が 405 以外の場合、およびどちらの情報源からも許可メソッドを解決できなかった場合は何もしません。
//
// RFC 9110 §15.5.6 は 405 に Allow を MUST としますが、これを付与する [echo] の methodNotAllowedHandler は
// 405 を短絡させるミドルウェア（OpenAPI バリデーション）の下流にあり、到達しない経路があります。
// そのため送出元を問わずここで補完します。
func setAllowHeader(c *echo.Context, policy AllowPolicy, status int) {
	if status != http.StatusMethodNotAllowed {
		return
	}

	allow := allowFromEchoRouter(c)
	if allow == "" {
		allow = policy.Allow(c.Request())
	}
	if allow == "" {
		return
	}

	c.Response().Header().Set(echo.HeaderAllow, allow)
}

// allowFromEchoRouter は、Echo のルータが解決した許可メソッド一覧を返します。解決していない場合は空文字を返します。
//
// Echo のルータが 405 と判断した場合にのみ [echo.ContextKeyHeaderAllow] を解決するため、値は常に得られるとは限りません。
// 静的パスと可変パスが重なる位置（`/v1/users/me` と `/v1/users/{userId}`）では、静的パス側に無いメソッドが
// 可変パス側のハンドラへ解決してしまい、405 と判断するのは OpenAPI のルータだけになります。
func allowFromEchoRouter(c *echo.Context) string {
	allow, ok := c.Get(echo.ContextKeyHeaderAllow).(string)
	if !ok {
		return ""
	}
	return allow
}
