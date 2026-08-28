package observability

import (
	"context"
	echootel "github.com/labstack/echo-opentelemetry"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ミドルウェアが生成される", func(t *testing.T) {
			t.Parallel()

			mw := Middleware()
			assert.NotNil(t, mw)
		})
	})
}

func TestPassthroughMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("後段ハンドラへ素通しし200を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			e.Use(PassthroughMiddleware())
			e.GET("/", func(c *echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	})
}

func TestMiddleware_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録ルートは404を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			e.Use(Middleware())

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)
		})

		t.Run("span属性にqueryの値を載せない", func(t *testing.T) {
			t.Parallel()

			// 秘匿すべき資格情報（stream ticket）は query で運ばれる（ADR-0074）。ミドルウェアが記録する span 属性は
			// url.path までで url.query を持たないことを、属性名の許可リストではなく値の全走査で固定する
			// （upstream が属性を増やしても、生値が現れた時点で落ちる）。
			exporter := &spanCollector{}
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
			t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })

			e := echo.New()
			e.Use(echootel.NewMiddlewareWithConfig(echootel.Config{TracerProvider: tp}))
			e.GET("/v1/streams/:destination", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/streams/s?ticket=raw-secret-value", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.NotEmpty(t, exporter.spans)
			for _, span := range exporter.spans {
				for _, attr := range span.Attributes() {
					assert.NotContains(t, attr.Value.Emit(), "raw-secret-value", "span 属性 %s に query の値が載っている", attr.Key)
				}
			}
		})
	})
}

// spanCollector は、終了した span をそのまま保持する同期 exporter です。
type spanCollector struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (c *spanCollector) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.spans = append(c.spans, spans...)
	return nil
}

func (c *spanCollector) Shutdown(context.Context) error { return nil }
