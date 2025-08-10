package server

import (
	"errors"
	"net/http"

	errorresponse "boilerplate-go/internal/controller/handler/error/response"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const (
	lowerBoundHTTPStatus = 400
	upperBoundHTTPStatus = 600

	errorLevelBoundHTTPStatus   = 500
	warningLevelBoundHTTPStatus = 400
)

// NewHTTPErrorHandler は、EchoのHTTPエラーハンドラーを生成します。
func NewHTTPErrorHandler(logger *zap.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		resp := normalizeHTTPError(err, getRequestID(c))
		if !c.Response().Committed {
			if err = c.JSON(resp.HTTPStatus, resp.ErrorResponse); err != nil {
				logger.Error("failed to write error response", zap.Error(err), zap.String("request_id", resp.RequestID))
				c.Response().WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		logHTTPError(logger, c, resp)
	}
}

// normalizeHTTPError は、HTTPエラーを正規化し、エラーレスポンスを生成します。
// リクエストIDを付与し、エラーの詳細を含めます。
func normalizeHTTPError(err error, requestID string) *errorresponse.HTTPErrorResponse {
	var he *errorresponse.HTTPErrorResponse
	if errors.As(err, &he) {
		if !isErrorStatus(he.HTTPStatus) {
			res := errorresponse.New(http.StatusInternalServerError, he.Internal)
			res.Details = he.Details
			res.RequestID = requestID
			return res
		}
		cp := *he
		cp.RequestID = requestID
		return &cp
	}

	var ehe *echo.HTTPError
	if errors.As(err, &ehe) {
		status := ehe.Code
		if !isErrorStatus(status) {
			status = http.StatusInternalServerError
		}
		res := errorresponse.New(status, err)
		res.RequestID = requestID
		return res
	}

	res := errorresponse.New(http.StatusInternalServerError, err)
	res.RequestID = requestID
	return res
}

// logHTTPError は、HTTPエラーをログに記録します。
func logHTTPError(logger *zap.Logger, c echo.Context, he *errorresponse.HTTPErrorResponse) {
	fields := []zap.Field{
		zap.Int("status", he.HTTPStatus),
		zap.String("method", c.Request().Method),
		zap.String("path", c.Request().URL.Path),
		zap.String("remote_ip", c.RealIP()),
		zap.String("request_id", he.RequestID),
		zap.String("error_code", he.Code),
		zap.String("error_message", he.Message),
	}
	if he.Details != nil {
		fields = append(fields, zap.Strings("error_details", *he.Details))
	}
	switch {
	case he.HTTPStatus >= errorLevelBoundHTTPStatus:
		fields = append(fields, zap.NamedError("stack", he.Internal))
		logger.Error("http_error", fields...)
	case he.HTTPStatus >= warningLevelBoundHTTPStatus:
		logger.Warn("http_error", fields...)
	default:
		logger.Info("http_error", fields...)
	}
}

// getRequestID は、EchoのコンテキストからリクエストIDを取得します。
func getRequestID(c echo.Context) string { return c.Request().Header.Get(echo.HeaderXRequestID) }

// isErrorStatus は、HTTPステータスコードがエラー範囲（400〜599）にあるかをチェックします。
func isErrorStatus(s int) bool { return lowerBoundHTTPStatus <= s && s < upperBoundHTTPStatus }
