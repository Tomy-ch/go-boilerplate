// Package errorhandler は、HTTPエラーハンドリングに関する機能を提供します。
package errorhandler

import (
	"net/http"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/error/response"
	"boilerplate-go/internal/controller/httpstack/requestid"
	"boilerplate-go/internal/controller/server"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"
	"boilerplate-go/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

const (
	lowerBoundHTTPStatus      = 400
	upperBoundHTTPStatus      = 600
	errorLevelBoundHTTPStatus = 500

	errHandlerKey = "http_error_handler"
)

// New は、EchoのHTTPエラーハンドラーを生成します。
func New(e *echo.Echo, log logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig) {
	e.HTTPErrorHandler = NewHTTPErrorHandler(log, lf, obsCfg)
}

// NewHTTPErrorHandler は、EchoのHTTPエラーハンドラーを生成します。
func NewHTTPErrorHandler(logger logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		handleHTTPError(c, logger, lf, obsCfg, err)
	}
}

// handleHTTPError は、HTTPエラーを処理し、適切なレスポンスをクライアントに返します。
func handleHTTPError(c echo.Context, logger logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig, err error) {
	if v, ok := c.Get(errHandlerKey).(bool); ok && v {
		return
	}
	c.Set(errHandlerKey, true)

	resp := normalizeHTTPError(err, requestid.GetRequestIDFromResponse(c))

	if !c.Response().Committed {
		if writeErr := writeErrorResponse(c, resp); writeErr != nil {
			req := c.Request()
			traceCtx := observability.ExtractTraceContext(req.Context())
			reqIn := logging.HTTPRequestLogInput{
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
				QueryParams:   server.ExtractQueryParams(c),
				PathParams:    server.ExtractPathParams(c),
				TraceID:       traceCtx.TraceID(),
				SpanID:        traceCtx.SpanID(),
			}
			writeErrFields := []*logging.Field{logging.String(logging.InternalErrorKey, writeErr.Error())}
			fields := append(lf.BuildHTTPRequestFields(reqIn), writeErrFields...)
			logger.Named("errorhandler.handleHTTPError").Error("failed to write error response", fields...)
			c.Response().WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	logHTTPError(c, logger, lf, obsCfg, resp)
}

// writeErrorResponse は、エラーレスポンスをクライアントに書き込みます。
func writeErrorResponse(c echo.Context, resp *response.HTTPErrorResponse) error {
	return c.JSON(resp.HTTPStatus, resp.ErrorResponse)
}

// normalizeHTTPError は、HTTPエラーを正規化し、エラーレスポンスを生成します。
// リクエストIDを付与し、エラーの詳細を含めます。
func normalizeHTTPError(
	err error,
	requestID string,
) *response.HTTPErrorResponse {
	var he *response.HTTPErrorResponse
	if xerrors.As(err, &he) {
		if !isErrorStatus(he.HTTPStatus) {
			res := response.NewHTTPErrorFromAppError(he.Internal)
			if he.Details != nil {
				res.Details = he.Details
			}
			res.RequestId = requestID
			return res
		}
		he.RequestId = requestID
		return he
	}

	var ehe *echo.HTTPError
	if xerrors.As(err, &ehe) {
		if res := normalizeOpenAPIError(ehe); res != nil {
			res.RequestId = requestID
			return res
		}
		if res := normalizeEchoHTTPError(ehe); res != nil {
			res.RequestId = requestID
			return res
		}
	}

	res := response.NewHTTPErrorFromAppError(err)
	res.RequestId = requestID
	return res
}

// httpErrorField は、HTTPエラーに関するログフィールドを生成します。
func httpErrorField(
	c echo.Context,
	lf logging.LogFieldBuilder,
	he *response.HTTPErrorResponse,
) []*logging.Field {
	req := c.Request()
	traceCtx := observability.ExtractTraceContext(req.Context())
	fields := []*logging.Field{
		logging.Int(logging.StatusKey, he.HTTPStatus),
		logging.String(logging.ErrorCodeKey, he.Code),
		logging.String(logging.ErrorMessageKey, he.Message),
		logging.String(logging.RequestIDKey, he.RequestId),
	}
	reqIn := logging.HTTPRequestLogInput{
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
		QueryParams:   server.ExtractQueryParams(c),
		PathParams:    server.ExtractPathParams(c),
		TraceID:       traceCtx.TraceID(),
		SpanID:        traceCtx.SpanID(),
	}
	fields = append(fields, lf.BuildHTTPRequestFields(reqIn)...)
	if he.Details != nil {
		fields = append(fields, logging.Strings(logging.ErrorDetails, *he.Details))
	}
	if he.Internal != nil {
		additionalFields := []*logging.Field{
			logging.String(logging.InternalErrorKey, he.Internal.Error()),
			logging.String(logging.InternalStackTraceKey, xerrors.StackTrace(he.Internal)),
		}
		fields = append(fields, additionalFields...)
	}
	return fields
}

// logHTTPError は、HTTPエラーをログに記録します。
func logHTTPError(
	c echo.Context,
	logger logging.Logger,
	lf logging.LogFieldBuilder,
	obsCfg *config.ObservabilityConfig,
	he *response.HTTPErrorResponse,
) {
	if !obsCfg.TargetStatusCodeSet()[he.HTTPStatus] {
		return
	}
	fields := httpErrorField(c, lf, he)
	switch {
	case he.HTTPStatus >= errorLevelBoundHTTPStatus:
		logger.Error("errorhandler.server_error", fields...)
	default:
		logger.Warn("errorhandler.client_error", fields...)
	}
}

// isErrorStatus は、HTTPステータスコードがエラー範囲（400〜599）にあるかをチェックします。
func isErrorStatus(
	s int,
) bool {
	return lowerBoundHTTPStatus <= s && s < upperBoundHTTPStatus
}
