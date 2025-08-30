package logging

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Test_buildFields(t *testing.T) {
	t.Parallel()

	expectedMethod := http.MethodGet
	expectedURI := "/test"
	expectedStatus := http.StatusOK
	expectedLatency := 100 * time.Millisecond
	expectedRemoteIP := "192.0.2.1"
	expectedTID := trace.TraceID{0x01, 0x02, 0x03}
	expectedSID := trace.SpanID{0x04, 0x05, 0x06}
	ipWithPort := expectedRemoteIP + ":12345"

	// echo.Contextを生成するヘルパー関数
	newContext := func() echo.Context {
		e := echo.New()
		req := httptest.NewRequest(expectedMethod, expectedURI, nil)
		req.RemoteAddr = ipWithPort
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    expectedTID,
			SpanID:     expectedSID,
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(req.Context(), spanCtx)
		c.SetRequest(req.WithContext(ctx))

		rec.WriteHeader(expectedStatus)
		c.Response().Status = expectedStatus

		return c
	}

	t.Run("エラーがある場合のフィールド生成", func(t *testing.T) {
		t.Parallel()

		c := newContext()

		expected := []zap.Field{
			zap.String("method", expectedMethod),
			zap.String("uri", expectedURI),
			zap.Int("status", expectedStatus),
			zap.Duration("latency", expectedLatency),
			zap.String("remote_ip", expectedRemoteIP),
			zap.String("trace_id", expectedTID.String()),
			zap.String("span_id", expectedSID.String()),
		}

		actual := buildFields(c, expectedLatency)

		require.Equal(t, expected, actual)
	})
}

func Test_logRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		status          int
		expectedLevel   string
		expectedMessage string
	}{
		{
			name:            "500 Internal Server Error→Errorログ",
			status:          500,
			expectedLevel:   "error",
			expectedMessage: "server error",
		},
		{
			name:            "404 Not Found →Warnログ",
			status:          404,
			expectedLevel:   "warn",
			expectedMessage: "client error",
		},
		{
			name:            "400 Bad Request→Warnログ",
			status:          400,
			expectedLevel:   "warn",
			expectedMessage: "client error",
		},
		{
			name:            "200 OK→Infoログ",
			status:          200,
			expectedLevel:   "info",
			expectedMessage: "request handled",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := newTestLogger(&buf)
			c := newTestContext(tt.status)

			logRequest(c, logger, nil)

			out := buf.String()
			require.Contains(t, out, `"level":"`+tt.expectedLevel+`"`)
			require.Contains(t, out, `"msg":"`+tt.expectedMessage+`"`)
		})
	}
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

func newTestContext(status int) echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	rec.WriteHeader(status)

	c := e.NewContext(req, rec)
	c.Response().Status = status
	return c
}
