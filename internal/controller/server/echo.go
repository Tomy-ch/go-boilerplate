package server

import (
	"time"

	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v5"
)

// BuildHTTPRequestLogInput は、Echo コンテキストから HTTP リクエストのログ入力を組み立てます（エラー/リカバリ経路の共通生成点）。
// eventType には呼び出し経路に応じたイベント種別（logging.EventTypeError / EventTypePanic 等）を渡す。
func BuildHTTPRequestLogInput(c *echo.Context, eventType string) logging.HTTPRequestLogInput {
	req := c.Request()
	return logging.HTTPRequestLogInput{
		EventType:     eventType,
		EventAt:       time.Now(),
		Method:        req.Method,
		Path:          c.Path(),
		URI:           req.RequestURI,
		RemoteIP:      c.RealIP(),
		Host:          req.Host,
		Scheme:        c.Scheme(),
		Proto:         req.Proto,
		UserAgent:     req.UserAgent(),
		ContentType:   req.Header.Get(echo.HeaderContentType),
		ContentLength: req.ContentLength,
		QueryParams:   ExtractQueryParams(c),
		PathParams:    ExtractPathParams(c),
	}
}

// ExtractPathParams は、Echoコンテキストからパスパラメータを抽出します。
func ExtractPathParams(c *echo.Context) map[string]string {
	values := c.PathValues()
	m := make(map[string]string, len(values))
	for _, v := range values {
		m[v.Name] = v.Value
	}
	return m
}

// ExtractQueryParams は、Echoコンテキストからクエリパラメータを抽出します。
func ExtractQueryParams(c *echo.Context) map[string][]string {
	// URL.Query() は呼び出し毎に新規 map を返すため、防御的コピーは不要。
	return c.Request().URL.Query()
}

// ResponseOf は、Echo コンテキストから [echo.Response] を取り出します。
// Echo v5 の Context.Response() は http.ResponseWriter を返すため、
// ステータスや Before/After フックを扱うにはこの unwrap が要ります。
// レスポンスライタが Echo のレスポンスへ辿れない場合は nil を返します。
func ResponseOf(c *echo.Context) *echo.Response {
	res, err := echo.UnwrapResponse(c.Response())
	if err != nil {
		return nil
	}
	return res
}
