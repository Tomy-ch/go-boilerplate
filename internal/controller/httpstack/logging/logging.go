// Package logging は、Echoフレームワークのミドルウェアとしてリクエストのログの出力を提供します。
package logging

import (
	"time"

	"boilerplate-go/internal/controller"
	"boilerplate-go/internal/controller/httpstack/requestid"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type log struct {
	c        echo.Context
	lf       logging.LogFields
	traceCtx observability.TraceContext
}

// Middleware は、Echoフレームワークのミドルウェアで、リクエストのログを出力します。
func Middleware(logger *zap.Logger, lf logging.LogFields) echo.MiddlewareFunc {
	return loggingMiddleware(logger, lf)
}

// loggingMiddleware は、リクエストのログを出力するミドルウェアを返します。
func loggingMiddleware(logger *zap.Logger, lf logging.LogFields) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			l := log{
				c:        c,
				lf:       lf,
				traceCtx: observability.ExtractSpan(c.Request().Context()),
			}

			reqFields := l.buildRequestLogFields()
			logger.Named("http.request").Info("request received", reqFields...)

			c.Response().After(func() {
				latency := time.Since(start)

				fields := l.buildResponseLogFields(latency)
				logger.Named("http.response").Info("request handled", fields...)
			})

			return next(c)
		}
	}
}

// buildRequestLogFields は、リクエストの情報を含むzap.Fieldのスライスを生成します。
func (l log) buildRequestLogFields() []zap.Field {
	req := l.c.Request()
	reqIn := logging.HTTPRequestLogInput{
		Method:        req.Method,
		URI:           req.RequestURI,
		Path:          req.URL.Path,
		RemoteIP:      req.RemoteAddr,
		Host:          req.Host,
		Scheme:        req.URL.Scheme,
		Proto:         req.Proto,
		UserAgent:     req.UserAgent(),
		ContentType:   req.Header.Get(echo.HeaderContentType),
		ContentLength: req.ContentLength,
		TraceID:       l.traceCtx.TraceID(),
		SpanID:        l.traceCtx.SpanID(),
		PathParams:    controller.ExtractPathParams(l.c),
		QueryParams:   controller.ExtractQueryParams(l.c),
	}

	return l.lf.BuildHTTPRequestFields(reqIn)
}

// buildResponseLogFields は、リクエストの情報を含むzap.Fieldのスライスを生成します。
func (l log) buildResponseLogFields(latency time.Duration) []zap.Field {
	req := l.c.Request()
	res := l.c.Response()
	resIn := logging.HTTPResponseLogInput{
		Method:    req.Method,
		Path:      req.URL.Path,
		URI:       req.RequestURI,
		Status:    res.Status,
		Latency:   latency,
		RequestID: requestid.GetRequestIDFromResponse(l.c),
		TraceID:   l.traceCtx.TraceID(),
		SpanID:    l.traceCtx.SpanID(),
	}
	return l.lf.BuildHTTPResponseFields(resIn)
}
