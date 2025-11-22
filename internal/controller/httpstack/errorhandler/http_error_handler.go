// Package errorhandler は、HTTPエラーハンドリングに関する機能を提供します。
package errorhandler

import (
	"errors"
	"fmt"
	"net/http"

	"boilerplate-go/internal/controller/error/response"
	"boilerplate-go/internal/controller/httpstack/requestid"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const (
	lowerBoundHTTPStatus = 400
	upperBoundHTTPStatus = 600

	errorLevelBoundHTTPStatus   = 500
	warningLevelBoundHTTPStatus = 400
)

// New は、EchoのHTTPエラーハンドラーを生成します。
func New(e *echo.Echo, z *zap.Logger) {
	e.HTTPErrorHandler = NewHTTPErrorHandler(z)
}

// NewHTTPErrorHandler は、EchoのHTTPエラーハンドラーを生成します。
func NewHTTPErrorHandler(logger *zap.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		resp := normalizeHTTPError(err, requestid.GetRequestIDFromResponse(c))
		if !c.Response().Committed {
			if err = c.JSON(resp.HTTPStatus, resp.ErrorResponse); err != nil {
				logger.Error(
					"failed to write error response",
					zap.Error(err),
					zap.String("request_id", resp.RequestId),
				)
				c.Response().WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		logHTTPError(logger, c, resp)
	}
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
	he *response.HTTPErrorResponse,
) []zap.Field {
	fields := []zap.Field{
		zap.Int("status", he.HTTPStatus),
		zap.String("method", c.Request().Method),
		zap.String("path", c.Request().URL.Path),
		zap.String("remote_ip", c.RealIP()),
		zap.String("request_id", he.RequestId),
		zap.String("error_code", he.Code),
		zap.String("error_message", he.Message),
	}
	if he.Details != nil {
		fields = append(fields, zap.Strings("error_details", *he.Details))
	}
	if he.Internal != nil {
		fields = append(fields, zap.String("internal_error", fmt.Sprintf("%v", he.Internal)))
	}
	return fields
}

// logHTTPError は、HTTPエラーをログに記録します。
func logHTTPError(
	logger *zap.Logger,
	c echo.Context,
	he *response.HTTPErrorResponse,
) {
	fields := httpErrorField(c, he)
	switch {
	case he.HTTPStatus >= errorLevelBoundHTTPStatus:
		logger.Error("http_error", fields...)
	case he.HTTPStatus >= warningLevelBoundHTTPStatus:
		logger.Warn("http_error", fields...)
	default:
		logger.Info("http_error", fields...)
	}
}

// isErrorStatus は、HTTPステータスコードがエラー範囲（400〜599）にあるかをチェックします。
func isErrorStatus(
	s int,
) bool {
	return lowerBoundHTTPStatus <= s && s < upperBoundHTTPStatus
}
