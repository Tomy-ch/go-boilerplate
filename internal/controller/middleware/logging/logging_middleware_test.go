package logging

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	ctxHelper "boilerplate-go/internal/controller/ctxhelper"
	expectedErrors "boilerplate-go/internal/domain/expectederrors"
	testUtil "boilerplate-go/internal/testutil"
	errUtil "boilerplate-go/pkg/errutil"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestMain(m *testing.M) {
	testUtil.RunWithTestSetup(m)
}

func Test_buildFields(t *testing.T) {
	t.Parallel()

	expectedMethod := http.MethodGet
	expectedURI := "/test"
	expectedStatus := http.StatusOK
	expectedLatency := 100 * time.Millisecond
	expectedRemoteIP := "192.0.2.1"
	expectedErr := errors.New("test error")
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

	t.Run("エラーがない場合のフィールド生成", func(t *testing.T) {
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

		actual := buildFields(c, expectedLatency, nil)

		assert.Equal(t, expected, actual)
	})

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
			zap.Error(expectedErr),
		}

		actual := buildFields(c, expectedLatency, expectedErr)

		assert.Equal(t, expected, actual)
	})
}

func Test_logRequest(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		notFound        expectedErrors.NotFoundCause
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
			name:            "404 Not Found (予期された)→Infoログ(expected not found)",
			status:          404,
			notFound:        expectedErrors.NotFoundCauseDB,
			expectedLevel:   "info",
			expectedMessage: "expected not found",
		},
		{
			name:            "404 Not Found (予期されていない)→Warnログ",
			status:          404,
			notFound:        expectedErrors.NotFoundCause("unexpected"),
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
			var buf bytes.Buffer
			logger := newTestLogger(&buf)
			c := newTestContext(tt.status)

			if tt.notFound != "" {
				ctxHelper.SetNotFoundToEcho(c, tt.notFound)
			}

			logRequest(c, logger, nil)

			out := buf.String()
			assert.Contains(t, out, `"level":"`+tt.expectedLevel+`"`)
			assert.Contains(t, out, `"msg":"`+tt.expectedMessage+`"`)
		})
	}
}

func Test_logErrorInDev(t *testing.T) {
	xerrs := errUtil.CockroachDBError{}

	cases := []struct {
		name       string
		appEnv     string
		err        error
		wantOutput bool
	}{
		{
			name:       "開発環境かつエラーありである場合、ログ出力される",
			appEnv:     config.DevelopmentMode,
			err:        xerrs.New("error"),
			wantOutput: true,
		},
		{
			name:       "開発環境だがエラーがnilである場合、ログ出力されない",
			appEnv:     config.DevelopmentMode,
			err:        nil,
			wantOutput: false,
		},
		{
			name:       "本番環境かつエラーありである場合、ログ出力されない",
			appEnv:     config.ProductionMode,
			err:        xerrs.New("error"),
			wantOutput: false,
		},
		{
			name:       "本番環境かつエラーなしである場合、ログ出力されない",
			appEnv:     config.ProductionMode,
			err:        nil,
			wantOutput: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			const envKey = "APP_ENV"

			t.Setenv(envKey, tt.appEnv)

			var buf bytes.Buffer
			logger := newTestLogger(&buf)

			cfg, err := config.New()
			if err != nil {
				panic("failed to create test config: " + err.Error())
			}

			logErrorInDev(logger, cfg, tt.err)

			out := buf.String()
			if tt.wantOutput {
				assert.Contains(t, out, `"msg":"\n`+tt.err.Error()+`"`)
			} else {
				assert.Equal(t, "", out)
			}
			assert.NoError(t, err)
		})
	}
}

func Test_isExpectedNotFound(t *testing.T) {
	t.Parallel()

	t.Run("contextに値が存在しない場合はfalse", func(t *testing.T) {
		t.Parallel()

		c := newTestEchoContext()
		ok := isExpectedNotFound(c)
		assert.False(t, ok)
	})

	t.Run("contextに未定義のNotFoundCauseがある場合はfalse", func(t *testing.T) {
		t.Parallel()

		c := newTestEchoContext()
		ctxHelper.SetNotFoundToEcho(c, expectedErrors.NotFoundCause("unexpected"))
		ok := isExpectedNotFound(c)
		assert.False(t, ok)
	})

	t.Run("contextに定義済みのNotFoundCauseがある場合はtrue", func(t *testing.T) {
		t.Parallel()

		c := newTestEchoContext()
		ctxHelper.SetNotFoundToEcho(c, expectedErrors.NotFoundCauseDB)
		ok := isExpectedNotFound(c)
		assert.True(t, ok)
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

func newTestContext(status int) echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	rec.WriteHeader(status)

	c := e.NewContext(req, rec)
	c.Response().Status = status
	return c
}

func newTestEchoContext() echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}
