// Package logging は、Echoフレームワークのミドルウェアとしてリクエストのログの出力を提供します。
package logging

import (
	"time"

	"go-boilerplate/internal/controller/httpstack/ops"
	"go-boilerplate/internal/controller/httpstack/requestid"
	"go-boilerplate/internal/controller/server"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v5"
)

type requestLog struct {
	c   *echo.Context
	res *echo.Response
	lf  logging.LogFieldBuilder
}

// Middleware は、Echoフレームワークのミドルウェアで、リクエストのログを出力します。
// ただし /health, /metrics 等の運用系エンドポイントはログ出力をスキップします。
// レスポンスを取り出せない場合はログを出さず素通しします。
func Middleware(logger logging.Logger, lf logging.LogFieldBuilder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if ops.IsOpsPath(c.Request().URL.Path) {
				return next(c)
			}

			res := server.ResponseOf(c)
			if res == nil {
				return next(c)
			}

			start := time.Now()
			ctx := c.Request().Context()

			l := requestLog{
				c:   c,
				res: res,
				lf:  lf,
			}

			reqFields := l.buildRequestLogFields(start)
			logger.Named("http.request").Info(ctx, "request received", reqFields...)

			res.After(func() {
				end := time.Now()
				latency := end.Sub(start)

				fields := l.buildResponseLogFields(end, latency)
				resLogger := logger.Named("http.response")
				if res.Status >= MinStatusError {
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
// eventAt はレスポンス完了時刻で、呼び出し元が単一ソースとして確定した値を注入します。
func (l requestLog) buildResponseLogFields(eventAt time.Time, latency time.Duration) []*logging.Field {
	req := l.c.Request()
	resIn := logging.HTTPResponseLogInput{
		EventAt:   eventAt,
		Method:    req.Method,
		Path:      req.URL.Path,
		URI:       req.RequestURI,
		Status:    l.res.Status,
		Latency:   latency,
		RequestID: requestid.GetRequestIDFromResponse(l.c),
	}
	return l.lf.BuildHTTPResponseFields(resIn)
}
