// Package logging は、Echoフレームワークのミドルウェアとしてリクエストのログの出力を提供します。
package logging

import (
	"time"

	"go-boilerplate/internal/controller/httpstack/ops"
	"go-boilerplate/internal/controller/httpstack/requestid"
	"go-boilerplate/internal/controller/server"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v4"
)

type requestLog struct {
	c  echo.Context
	lf logging.LogFieldBuilder
}

// Middleware は、Echoフレームワークのミドルウェアで、リクエストのログを出力します。
// ただし /health, /metrics 等の運用系エンドポイントはログ出力をスキップします。
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
			ctx := c.Request().Context()

			l := requestLog{
				c:  c,
				lf: lf,
			}

			reqFields := l.buildRequestLogFields(start)
			logger.Named("http.request").Info(ctx, "request received", reqFields...)

			c.Response().After(func() {
				latency := time.Since(start)

				fields := l.buildResponseLogFields(latency)
				resLogger := logger.Named("http.response")
				if c.Response().Status >= MinStatusError {
					resLogger.Error(ctx, "request handled", fields...)
				} else {
					resLogger.Info(ctx, "request handled", fields...)
				}
			})

			return next(c)
		}
	}
}

// buildRequestLogFields は、リクエストの情報を含むFieldのスライスを生成します。
func (l requestLog) buildRequestLogFields(start time.Time) []*logging.Field {
	req := l.c.Request()
	reqIn := logging.HTTPRequestLogInput{
		EventType:     logging.EventTypeStart,
		EventAt:       start,
		Method:        req.Method,
		URI:           req.RequestURI,
		Path:          req.URL.Path,
		RemoteIP:      req.RemoteAddr,
		Host:          req.Host,
		Scheme:        l.c.Scheme(),
		Proto:         req.Proto,
		UserAgent:     req.UserAgent(),
		ContentType:   req.Header.Get(echo.HeaderContentType),
		ContentLength: req.ContentLength,
		PathParams:    server.ExtractPathParams(l.c),
		QueryParams:   server.ExtractQueryParams(l.c),
	}

	return l.lf.BuildHTTPRequestFields(reqIn)
}

// buildResponseLogFields は、レスポンスの情報を含むFieldのスライスを生成します。
func (l requestLog) buildResponseLogFields(latency time.Duration) []*logging.Field {
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
	}
	return l.lf.BuildHTTPResponseFields(resIn)
}
