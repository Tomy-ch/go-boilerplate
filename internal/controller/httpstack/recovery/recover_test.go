package recovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/httpstack/errorhandler"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのミドルウェアを返す", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			lf := logging.NewTestLogFieldBuilder(t)
			logger := logging.NewTestLogger(t)

			require.NotNil(t, Middleware(logger, lf, appCfg))
		})

		t.Run("パニックが無ければ後続の戻り値をそのまま返しログも残さない", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			lf := logging.NewTestLogFieldBuilder(t)
			obsLogger, observed := logging.NewObservedTestLogger(t)

			wantErr := xerrors.New("handler error")

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ok", nil)
			c := e.NewContext(req, httptest.NewRecorder())

			handler := Middleware(obsLogger, lf, appCfg)(func(_ *echo.Context) error { return wantErr })

			require.ErrorIs(t, handler(c), wantErr)
			assert.Equal(t, 0, observed.Len())
			recovered, _ := ctxhelper.GetRecoveredFromEcho(c)
			assert.False(t, recovered)
		})

		t.Run("パニックがリカバーされ可読スタックがログに残る", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			lf := logging.NewTestLogFieldBuilder(t)
			obsLogger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			e.Use(Middleware(obsLogger, lf, appCfg))
			e.GET("/panic", func(_ *echo.Context) error { panic("boom-panic") })

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/panic", nil)
			e.ServeHTTP(httptest.NewRecorder(), req)

			entries := observed.FilterMessage("panic recovered").All()
			require.Len(t, entries, 1)
			cm := entries[0].ContextMap()

			assert.Equal(t, logging.EventTypePanic, cm[logging.EventTypeKey])

			errStr, ok := cm[logging.InternalErrorKey].(string)
			require.True(t, ok)
			assert.Contains(t, errStr, "boom-panic")

			// 実ランタイムスタックが行配列で出力されること（Grafana 表示用に []string 化済み）。
			// zap observer は []any として保持するため、要素ごとに string にアサートする。
			stackLines, ok := cm[logging.InternalStackTraceKey].([]any)
			require.True(t, ok)
			require.NotEmpty(t, stackLines)
			first, ok := stackLines[0].(string)
			require.True(t, ok)
			assert.Contains(t, first, "goroutine")
		})

		t.Run("パニックでも500を返しログはmiddleware.recoverの1件のみになる", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			obsCfg := config.NewObservabilityConfig(cfg)
			lf := logging.NewTestLogFieldBuilder(t)
			obsLogger, observed := logging.NewObservedTestLogger(t)

			spec, err := validator.GetValidator()
			require.NoError(t, err)
			detailPolicy, err := errorhandler.NewOpenAPIDetailPolicy(spec)
			require.NoError(t, err)
			allowPolicy, err := errorhandler.NewOpenAPIAllowPolicy(spec)
			require.NoError(t, err)

			e := echo.New()
			errorhandler.New(e, errorhandler.Policies{Detail: detailPolicy, Allow: allowPolicy}, obsLogger, lf, obsCfg)
			e.Use(Middleware(obsLogger, lf, appCfg))
			e.GET("/panic", func(_ *echo.Context) error { panic("boom-panic") })

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/panic", nil)
			e.ServeHTTP(rec, req)

			// パニックでも 200 空ではなく 500 が返る。
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			// ログは middleware.recover の 1 件のみ（エラーハンドラの重複ログは出ない）。
			assert.Equal(t, 1, observed.Len())
			assert.Equal(t, 1, observed.FilterMessage("panic recovered").Len())
			assert.Equal(t, 0, observed.FilterMessage("errorhandler.server_error").Len())
		})

		t.Run("エラーハンドラへはスタックを含まない元のパニックエラーが渡る", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			lf := logging.NewTestLogFieldBuilder(t)
			obsLogger, _ := logging.NewObservedTestLogger(t)

			panicked := xerrors.New("boom-panic")

			e := echo.New()
			var got error
			e.HTTPErrorHandler = func(_ *echo.Context, err error) { got = err }
			e.Use(Middleware(obsLogger, lf, appCfg))
			e.GET("/panic", func(_ *echo.Context) error { panic(panicked) })

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/panic", nil)
			e.ServeHTTP(httptest.NewRecorder(), req)

			require.ErrorIs(t, got, panicked)
			var pse *middleware.PanicStackError
			assert.False(t, xerrors.As(got, &pse), "スタックを抱えたままのエラーは伝播しない")
		})

		t.Run("http.ErrAbortHandlerのパニックはリカバーせず再パニックしログも残さない", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			lf := logging.NewTestLogFieldBuilder(t)
			obsLogger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			e.Use(Middleware(obsLogger, lf, appCfg))
			e.GET("/abort", func(_ *echo.Context) error { panic(http.ErrAbortHandler) })

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/abort", nil)

			// echo の recover は http.ErrAbortHandler を握り潰さず再パニックする。
			assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
				e.ServeHTTP(rec, req)
			})
			// 再パニックされるためログ関数は呼ばれず "panic recovered" ログは出力されない。
			assert.Equal(t, 0, observed.FilterMessage("panic recovered").Len())
		})
	})
}

func Test_newPanicLogFunc(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)
	e := echo.New()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RemoteAddrがある場合、パニックをログしリカバリ済みを記録する", func(t *testing.T) {
			t.Parallel()

			obsLogger, observed := logging.NewObservedTestLogger(t)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/path", nil)
			req.RemoteAddr = "9.8.7.6:1234"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			newPanicLogFunc(obsLogger, lf)(c, xerrors.New("boom"), []byte("stack"))

			assert.Equal(t, 1, observed.FilterMessage("panic recovered").Len())
			recovered, _ := ctxhelper.GetRecoveredFromEcho(c)
			assert.True(t, recovered)
		})

		t.Run("X-Real-Ipヘッダがある場合、パニックをログしリカバリ済みを記録する", func(t *testing.T) {
			t.Parallel()

			obsLogger, observed := logging.NewObservedTestLogger(t)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/other", nil)
			req.Header.Set("X-Real-Ip", "10.0.0.1")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			newPanicLogFunc(obsLogger, lf)(c, xerrors.New("boom2"), []byte("stack2"))

			assert.Equal(t, 1, observed.FilterMessage("panic recovered").Len())
			recovered, _ := ctxhelper.GetRecoveredFromEcho(c)
			assert.True(t, recovered)
		})
	})
}

func Test_newRecoverConfig(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開発モードの場合、developmentConfigを返す", func(t *testing.T) {
			t.Parallel()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			appCfg.SetApplicationMode(t, config.DevelopmentMode)

			expected := developmentConfig()
			actual := newRecoverConfig(logger, appCfg)
			assert.Equal(t, expected, actual)
		})

		t.Run("本番モードの場合、productionConfigを返す", func(t *testing.T) {
			t.Parallel()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			appCfg.SetApplicationMode(t, config.ProductionMode)

			expected := productionConfig()
			actual := newRecoverConfig(logger, appCfg)
			assert.Equal(t, expected, actual)
		})

		t.Run("不明なモードの場合、warningを出してproductionConfigを返す", func(t *testing.T) {
			t.Parallel()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			appCfg.SetApplicationMode(t, "unknown-mode")

			expected := productionConfig()
			actual := newRecoverConfig(logger, appCfg)
			assert.Equal(t, expected, actual)
		})
	})
}

func Test_developmentConfig(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開発用設定が返る", func(t *testing.T) {
			t.Parallel()
			expected := middleware.RecoverConfig{
				StackSize:         10 << 10,
				DisableStackAll:   false,
				DisablePrintStack: false,
			}

			actual := developmentConfig()
			assert.Equal(t, expected, actual)
		})
	})
}

func Test_productionConfig(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本番用設定が返る", func(t *testing.T) {
			t.Parallel()
			expected := middleware.RecoverConfig{
				StackSize:         4 << 10,
				DisableStackAll:   true,
				DisablePrintStack: false,
			}

			actual := productionConfig()
			assert.Equal(t, expected, actual)
		})

		t.Run("本番設定でもruntimeスタックが捕捉される", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			handler := middleware.RecoverWithConfig(productionConfig())(
				func(_ *echo.Context) error { panic("prod-panic") },
			)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/panic", nil)
			c := e.NewContext(req, httptest.NewRecorder())

			var pse *middleware.PanicStackError
			require.ErrorAs(t, handler(c), &pse)
			// 本番設定(DisablePrintStack=false)でも runtime スタックが捕捉される。
			assert.Contains(t, string(pse.Stack), "goroutine")
		})
	})
}
