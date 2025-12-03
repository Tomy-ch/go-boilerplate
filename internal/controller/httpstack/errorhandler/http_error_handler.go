// Package errorhandler は、HTTPエラーハンドリングに関する機能を提供します。
package errorhandler

import (
	"errors"
	"net/http"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller"
	"boilerplate-go/internal/controller/error/response"
	"boilerplate-go/internal/controller/httpstack/requestid"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"
	"boilerplate-go/pkg/xerrors"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const (
	lowerBoundHTTPStatus = 400
	upperBoundHTTPStatus = 600

	errorLevelBoundHTTPStatus = 500
)

// New は、EchoのHTTPエラーハンドラーを生成します。
func New(e *echo.Echo, z *zap.Logger, lf logging.LogFields, obsCfg *config.ObservabilityConfig) {
	e.HTTPErrorHandler = NewHTTPErrorHandler(z, lf, obsCfg)
}

// NewHTTPErrorHandler は、EchoのHTTPエラーハンドラーを生成します。
func NewHTTPErrorHandler(logger *zap.Logger, lf logging.LogFields, obsCfg *config.ObservabilityConfig) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		handleHTTPError(c, logger, lf, obsCfg, err)
	}
}

// handleHTTPError は、HTTPエラーを処理し、適切なレスポンスをクライアントに返します。
func handleHTTPError(c echo.Context, logger *zap.Logger, lf logging.LogFields, obsCfg *config.ObservabilityConfig, err error) {
	resp := normalizeHTTPError(err, requestid.GetRequestIDFromResponse(c))

	if !c.Response().Committed {
		if writeErr := writeErrorResponse(c, resp); writeErr != nil {
			req := c.Request()
			traceCtx := observability.ExtractSpan(req.Context())
			reqIn := logging.HTTPRequestLogInput{
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
				QueryParams:   controller.ExtractQueryParams(c),
				PathParams:    controller.ExtractPathParams(c),
				TraceID:       traceCtx.TraceID(),
				SpanID:        traceCtx.SpanID(),
			}
			writeErrFields := []zap.Field{zap.String(logging.InternalErrorKey, writeErr.Error())}
			fields := append(lf.BuildHTTPRequestFields(reqIn), writeErrFields...)
			logger.Error("failed to write error response", fields...)
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
	if errors.As(err, &he) {
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
	if errors.As(err, &ehe) {
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
	lf logging.LogFields,
	he *response.HTTPErrorResponse,
) []zap.Field {
	req := c.Request()
	traceCtx := observability.ExtractSpan(req.Context())
	fields := []zap.Field{
		zap.Int(logging.StatusKey, he.HTTPStatus),
		zap.String(logging.ErrorCodeKey, he.Code),
		zap.String(logging.ErrorMessageKey, he.Message),
		zap.String(logging.RequestIDKey, he.RequestId),
	}
	reqIn := logging.HTTPRequestLogInput{
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
		QueryParams:   controller.ExtractQueryParams(c),
		PathParams:    controller.ExtractPathParams(c),
		TraceID:       traceCtx.TraceID(),
		SpanID:        traceCtx.SpanID(),
	}
	fields = append(fields, lf.BuildHTTPRequestFields(reqIn)...)
	if he.Details != nil {
		fields = append(fields, zap.Strings(logging.ErrorDetails, *he.Details))
	}
	if he.Internal != nil {
		additionalFields := []zap.Field{
			zap.String(logging.InternalErrorKey, he.Internal.Error()),
			zap.String(logging.InternalStackTraceKey, xerrors.StackTrace(he.Internal)),
		}
		fields = append(fields, additionalFields...)
	}
	return fields
}

// logHTTPError は、HTTPエラーをログに記録します。
func logHTTPError(
	c echo.Context,
	logger *zap.Logger,
	lf logging.LogFields,
	obsCfg *config.ObservabilityConfig,
	he *response.HTTPErrorResponse,
) {
	if !observability.ShouldLogWithSpan(c.Request().Context(), obsCfg) {
		return
	}
	fields := httpErrorField(c, lf, he)
	switch {
	case he.HTTPStatus >= errorLevelBoundHTTPStatus:
		logger.Error("server_error", fields...)
	default:
		logger.Warn("client_error", fields...)
	}
}

// isErrorStatus は、HTTPステータスコードがエラー範囲（400〜599）にあるかをチェックします。
func isErrorStatus(
	s int,
) bool {
	return lowerBoundHTTPStatus <= s && s < upperBoundHTTPStatus
}
