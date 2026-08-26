package skipper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	sk := New()
	exec := func(t *testing.T, path string) bool {
		t.Helper()
		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		return sk(c)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("/healthの場合はスキップ対象としてtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, exec(t, "/health"))
		})

		t.Run("業務エンドポイントの場合はfalseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, exec(t, "/v1/users"))
		})
	})
}
