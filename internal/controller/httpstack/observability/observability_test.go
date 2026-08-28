package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	obs "go-boilerplate/internal/observability"

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

		t.Run("span属性にqueryの値を載せない", func(t *testing.T) { //nolint:paralleltest // otel の global provider を差し替えるため

			// 秘匿すべき資格情報（stream ticket）は query で運ばれる（ADR-0074）。ミドルウェアが記録する span 属性は
			// url.path までで url.query を持たないことを、属性名の許可リストではなく値の全走査で固定する
			// （upstream が属性を増やしても、生値が現れた時点で落ちる）。
			recorded := obs.InstallRecordingTracerProvider(t)

			e := echo.New()
			e.Use(Middleware())
			e.GET("/v1/streams/:destination", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/streams/s?ticket=raw-secret-value", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			// 並行する別ケースの span が混ざりうるので、被験リクエストの span（url.path が一致するもの）に絞ってから走査する。
			var targets []string
			for _, span := range recorded() {
				matched := false
				for _, attr := range span.Attributes() {
					if attr.Key == "url.path" && attr.Value.AsString() == "/v1/streams/s" {
						matched = true
					}
				}
				if !matched {
					continue
				}
				targets = append(targets, span.Name())
				for _, attr := range span.Attributes() {
					assert.NotContainsf(t, attr.Value.AsString(), "raw-secret-value",
						"span 属性 %s に query の値が載っている", attr.Key)
				}
			}
			require.NotEmpty(t, targets, "被験リクエストの span が記録されていない")
		})
	})
}
