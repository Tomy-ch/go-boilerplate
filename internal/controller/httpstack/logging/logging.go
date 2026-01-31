// Package logging は、Echoフレームワークのミドルウェアとしてリクエストのログの出力を提供します。
package logging

import (
	"time"

	"boilerplate-go/internal/controller/httpstack/ops"
	"boilerplate-go/internal/controller/httpstack/requestid"
	"boilerplate-go/internal/controller/server"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
)

type log struct {
	c        echo.Context
	lf       logging.LogFieldBuilder
	traceCtx *observability.TraceContext
}

// Middleware は、Echoフレームワークのミドルウェアで、リクエストのログを出力します。
func Middleware(logger logging.Logger, lf logging.LogFieldBuilder) echo.MiddlewareFunc {
	return loggingMiddleware(logger, lf)
}

// loggingMiddleware は、リクエストのログを出力するミドルウェアを返します。
func loggingMiddleware(logger logging.Logger, lf logging.LogFieldBuilder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if ops.IsOpsPath(c.Request().URL.Path) {
				return next(c)
			}

			start := time.Now()

			l := log{
				c:        c,
				lf:       lf,
				traceCtx: observability.ExtractTraceContext(c.Request().Context()),
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

// buildRequestLogFields は、リクエストの情報を含むFieldのスライスを生成します。
func (l log) buildRequestLogFields() []*logging.Field {
	req := l.c.Request()
	reqIn := logging.HTTPRequestLogInput{
		EventAt:       time.Now(),
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
		PathParams:    server.ExtractPathParams(l.c),
		QueryParams:   server.ExtractQueryParams(l.c),
	}

	return l.lf.BuildHTTPRequestFields(reqIn)
}

// buildResponseLogFields は、リクエストの情報を含むFieldのスライスを生成します。
func (l log) buildResponseLogFields(latency time.Duration) []*logging.Field {
	req := l.c.Request()
	res := l.c.Response()
	resIn := logging.HTTPResponseLogInput{
		EventAt:   time.Now(),
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
