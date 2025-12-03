package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler/handlertest/testspan"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Test_requestLogMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	next := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/mw", nil)
	req.RemoteAddr = "203.0.113.5:45678"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := loggingMiddleware(logger, lf)(next)
	require.NoError(t, handler(c))

	// buf から request handled の行を探して JSON パース
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var reqLine, resLine string
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], `"msg":"request handled"`) {
			resLine = lines[i]
		}
		if strings.Contains(lines[i], `"msg":"request received"`) {
			reqLine = lines[i]
		}
	}
	require.NotEmpty(t, reqLine)
	require.NotEmpty(t, resLine)

	var gotReq map[string]any
	var gotRes map[string]any
	require.NoError(t, json.Unmarshal([]byte(reqLine), &gotReq))
	require.NoError(t, json.Unmarshal([]byte(resLine), &gotRes))

	require.Equal(t, http.MethodGet, fmt.Sprint(gotReq["method"]))
	require.Equal(t, "/mw", fmt.Sprint(gotReq["uri"]))

	// trace_id / span_id は存在すればゼロ値か空文字列
	if v, ok := gotRes["trace_id"]; ok {
		vs := fmt.Sprint(v)
		require.True(t, vs == "" || strings.Count(vs, "0") >= 8)
	}
	if v, ok := gotRes["span_id"]; ok {
		vs := fmt.Sprint(v)
		require.True(t, vs == "" || strings.Count(vs, "0") >= 4)
	}
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	require.NotNil(t, Middleware(logger, lf))
}

func Test_log_buildRequestLogFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/path?foo=bar", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("User-Agent", "ua-test")
	req.Host = "example.local"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("スパンなしはtrace/spanは含まれないリクエストログフィールドセットを返す", func(t *testing.T) {
		t.Parallel()

		tc := observability.TraceContext{}
		l := log{c: c, lf: lf, traceCtx: tc}
		fields := l.buildRequestLogFields()

		require.Contains(t, fields, zap.String(logging.MethodKey, http.MethodGet))
		require.Contains(t, fields, zap.String(logging.URIKey, "/path?foo=bar"))
		require.Contains(t, fields, zap.String(logging.PathKey, "/path"))
		require.Contains(t, fields, zap.String(logging.RemoteIPKey, req.RemoteAddr))
		require.Contains(t, fields, zap.String(logging.HostKey, req.Host))
		require.Contains(t, fields, zap.String(logging.UserAgentKey, "ua-test"))

		require.False(t, hasField(fields, logging.TraceIDKey))
		require.False(t, hasField(fields, logging.SpanIDKey))
	})

	t.Run("スパンありはtrace/spanが含まれるリクエストログフィールドセットを返す", func(t *testing.T) {
		t.Parallel()

		// リクエストコンテキストにテストスパンを埋め込む
		cWithSpan, end := testspan.StartTestSpanForEcho(t, c)
		defer end()

		tc := observability.ExtractSpan(cWithSpan.Request().Context())
		l := log{c: cWithSpan, lf: lf, traceCtx: tc}
		fields := l.buildRequestLogFields()

		require.Contains(t, fields, zap.String(logging.MethodKey, http.MethodGet))
		require.Contains(t, fields, zap.String(logging.URIKey, "/path?foo=bar"))

		require.True(t, hasField(fields, logging.TraceIDKey))
		require.True(t, hasField(fields, logging.SpanIDKey))
		require.NotEmpty(t, tc.TraceID)
		require.NotEmpty(t, tc.SpanID)
	})
}

func Test_log_buildResponseLogFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/resp", nil)
	req.RemoteAddr = "5.6.7.8:9000"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("スパンなしはtrace/spanは含まれないレスポンスログフィールドセットを返す", func(t *testing.T) {
		t.Parallel()

		expectedStatus := http.StatusCreated
		expectedRequestID := "req-123"

		c.Response().Status = expectedStatus
		c.Response().Header().Set("X-Request-Id", expectedRequestID)

		l := log{c: c, lf: lf, traceCtx: observability.TraceContext{}}
		fields := l.buildResponseLogFields(150 * time.Millisecond)

		require.Contains(t, fields, zap.Int(logging.StatusKey, expectedStatus))
		require.Contains(t, fields, zap.String(logging.RequestIDKey, expectedRequestID))

		require.False(t, hasField(fields, logging.TraceIDKey))
		require.False(t, hasField(fields, logging.SpanIDKey))
	})

	t.Run("スパンありはtrace/spanが含まれるレスポンスログフィールドセットを返す", func(t *testing.T) {
		t.Parallel()

		expectedStatus := http.StatusAccepted
		expectedRequestID := "req-accepted"

		cWithSpan, end := testspan.StartTestSpanForEcho(t, c)
		defer end()

		cWithSpan.Response().Status = expectedStatus
		cWithSpan.Response().Header().Set("X-Request-Id", expectedRequestID)

		tc := observability.ExtractSpan(cWithSpan.Request().Context())
		l := log{c: cWithSpan, lf: lf, traceCtx: tc}
		fields := l.buildResponseLogFields(20 * time.Millisecond)

		require.Contains(t, fields, zap.Int(logging.StatusKey, expectedStatus))
		require.Contains(t, fields, zap.String(logging.RequestIDKey, expectedRequestID))

		require.True(t, hasField(fields, logging.TraceIDKey))
		require.True(t, hasField(fields, logging.SpanIDKey))
	})
}

func newTestLogger(buf *bytes.Buffer) *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = ""
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(buf),
		zapcore.DebugLevel,
	)
	return zap.New(core)
}

// hasField は、fs に key を持つフィールドが含まれているかを返す。
func hasField(fs []zapcore.Field, key string) bool {
	for _, f := range fs {
		if f.Key == key {
			return true
		}
	}
	return false
}
