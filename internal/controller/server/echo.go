package server

import (
	"net/url"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v4"
)

// BuildHTTPRequestLogInput は、EchoコンテキストからHTTPリクエストのログ入力を組み立てます。
// エラーハンドラ・リカバリのように「エラー発生時点(time.Now)・解決済みパス(c.Path)・実IP(c.RealIP)」で
// ログを取りたい経路の共通生成点です（正常系の逐次ログは開始時刻起点のため対象外）。
func BuildHTTPRequestLogInput(c echo.Context) logging.HTTPRequestLogInput {
	req := c.Request()
	traceCtx := observability.ExtractTraceContext(req.Context())
	return logging.HTTPRequestLogInput{
		EventAt:       time.Now(),
		Method:        req.Method,
		Path:          c.Path(),
		URI:           req.RequestURI,
		RemoteIP:      c.RealIP(),
		Host:          req.Host,
		Scheme:        req.URL.Scheme,
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
	return cloneValues(c.Request().URL.Query())
}

// cloneValues は、url.Valuesのディープコピーを作成してmap[string][]stringとして返します。
func cloneValues(v url.Values) map[string][]string {
	if v == nil {
		return nil
	}
	m := make(map[string][]string, len(v))
	for k, vals := range v {
		cp := make([]string, len(vals))
		copy(cp, vals)
		m[k] = cp
	}
	return m
}
