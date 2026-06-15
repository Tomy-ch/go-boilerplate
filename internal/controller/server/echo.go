package server

import (
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v4"
)

// BuildHTTPRequestLogInput は、Echo コンテキストから HTTP リクエストのログ入力を組み立てます（エラー/リカバリ経路の共通生成点）。
// eventType には呼び出し経路に応じたイベント種別（logging.EventTypeError / EventTypePanic 等）を渡す。
func BuildHTTPRequestLogInput(c echo.Context, eventType string) logging.HTTPRequestLogInput {
	req := c.Request()
	traceCtx := observability.ExtractTraceContext(req.Context())
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
		TraceID:       traceCtx.TraceID(),
		SpanID:        traceCtx.SpanID(),
	}
}

// ExtractPathParams は、Echoコンテキストからパスパラメータを抽出します。
func ExtractPathParams(c echo.Context) map[string]string {
	m := make(map[string]string, len(c.ParamNames()))
	for _, name := range c.ParamNames() {
		m[name] = c.Param(name)
	}
	return m
}

// ExtractQueryParams は、Echoコンテキストからクエリパラメータを抽出します。
func ExtractQueryParams(c echo.Context) map[string][]string {
	// URL.Query() は呼び出し毎に新規 map を返すため、防御的コピーは不要。
	return c.Request().URL.Query()
}
