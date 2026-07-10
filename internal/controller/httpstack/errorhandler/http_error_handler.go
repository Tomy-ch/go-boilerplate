// Package errorhandler は、HTTPエラーハンドリングに関する機能を提供します。
package errorhandler

import (
	"net/http"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/error/response"
	"go-boilerplate/internal/controller/httpstack/requestid"
	"go-boilerplate/internal/controller/server"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

const (
	lowerBoundHTTPStatus      = 400
	upperBoundHTTPStatus      = 600
	errorLevelBoundHTTPStatus = 500
)

// New は、NewHTTPErrorHandler で生成したハンドラを Echo の HTTPErrorHandler として登録します。
func New(e *echo.Echo, log logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig) {
	e.HTTPErrorHandler = NewHTTPErrorHandler(log, lf, obsCfg)
}

// NewHTTPErrorHandler は、echo.HTTPErrorHandler を生成して返します。
func NewHTTPErrorHandler(logger logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		handleHTTPError(c, logger, lf, obsCfg, err)
	}
}

// handleHTTPError は、HTTPエラーを処理し、適切なレスポンスをクライアントに返します。
func handleHTTPError(c echo.Context, logger logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig, err error) {
	if handled, _ := ctxhelper.GetErrorHandledFromEcho(c); handled {
		return
	}
	ctxhelper.SetErrorHandledToEcho(c, true)

	resp := normalizeHTTPError(err, requestid.GetRequestIDFromResponse(c))

	if !c.Response().Committed {
		if writeErr := writeErrorResponse(c, resp); writeErr != nil {
			reqIn := server.BuildHTTPRequestLogInput(c, logging.EventTypeError)
			writeErrFields := []*logging.Field{logging.String(logging.InternalErrorKey, writeErr.Error())}
			fields := append(lf.BuildHTTPRequestFields(reqIn), writeErrFields...)
			logger.Named("errorhandler.handleHTTPError").Error(c.Request().Context(), "failed to write error response", fields...)
			// ヘッダ送出途中の失敗ではレスポンスが commit 済みになり得るため、未 commit 時のみ 500 を書く。
			if !c.Response().Committed {
				c.Response().WriteHeader(http.StatusInternalServerError)
			}
			return
		}
	}

	// リカバリ済みのパニックは middleware.recover が既にログ済みのため、二重ログを抑止する（500 応答は返す）。
	if recovered, _ := ctxhelper.GetRecoveredFromEcho(c); !recovered {
		logHTTPError(c, logger, lf, obsCfg, resp)
	}
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
		// ステータスがエラー域外(400〜599 外)の不正な HTTPErrorResponse は、Internal を真とみなして 500 系へ矯正する。
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
	fields := []*logging.Field{
		logging.Int(logging.StatusKey, he.HTTPStatus),
		logging.String(logging.ErrorCodeKey, he.Code),
		logging.String(logging.ErrorMessageKey, he.Message),
		logging.String(logging.RequestIDKey, he.RequestId),
	}
	fields = append(fields, lf.BuildHTTPRequestFields(server.BuildHTTPRequestLogInput(c, logging.EventTypeError))...)
	if he.Details != nil {
		fields = append(fields, logging.Strings(logging.ErrorDetailsKey, *he.Details))
	}
	if he.Internal != nil {
		additionalFields := []*logging.Field{
			logging.String(logging.InternalErrorKey, he.Internal.Error()),
			logging.Stacktrace(logging.InternalStackTraceKey, he.Internal),
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
	ctx := c.Request().Context()
	switch {
	case he.HTTPStatus >= errorLevelBoundHTTPStatus:
		logger.Error(ctx, "errorhandler.server_error", fields...)
	default:
		logger.Warn(ctx, "errorhandler.client_error", fields...)
	}
}

// isErrorStatus は、HTTPステータスコードがエラー範囲（400〜599）にあるかをチェックします。
func isErrorStatus(
	s int,
) bool {
	return lowerBoundHTTPStatus <= s && s < upperBoundHTTPStatus
}
