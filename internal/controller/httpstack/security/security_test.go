package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_buildSecureConfig(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定値を反映した SecureConfig を生成する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			secCfg := config.NewSecurityConfig(cfg)

			scfg := buildSecureConfig(secCfg)

			assert.Equal(t, secCfg.ContentTypeNosniff(), scfg.ContentTypeNosniff)
			assert.Equal(t, secCfg.ReferrerPolicy(), scfg.ReferrerPolicy)
			assert.Equal(t, secCfg.XFrameOptions(), scfg.XFrameOptions)
			assert.Equal(t, int(secCfg.HSTSMaxAge().Seconds()), scfg.HSTSMaxAge)
			assert.Equal(t, secCfg.HSTSExcludeSubdomains(), scfg.HSTSExcludeSubdomains)
			assert.Equal(t, secCfg.HSTSPreloadEnabled(), scfg.HSTSPreloadEnabled)
		})
	})
}

func Test_applyCORP(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("値が設定されていればヘッダーへ反映する", func(t *testing.T) {
			t.Parallel()

			h := http.Header{}
			applyCORP(h, "same-origin")

			assert.Equal(t, "same-origin", h.Get(headerCrossOriginResourcePolicy))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("値が空ならヘッダー自体を設定しない", func(t *testing.T) {
			t.Parallel()

			h := http.Header{}
			applyCORP(h, "")

			_, ok := h[headerCrossOriginResourcePolicy]
			assert.False(t, ok)
		})
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定されたセキュリティヘッダーをレスポンスへ付与する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			secCfg := config.NewSecurityConfig(cfg)

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := Middleware(secCfg)(func(c *echo.Context) error {
				return c.NoContent(http.StatusOK)
			})
			require.NoError(t, handler(c))

			require.NotEmpty(t, secCfg.CrossOriginResourcePolicy())
			assert.Equal(t, secCfg.CrossOriginResourcePolicy(), rec.Header().Get(headerCrossOriginResourcePolicy))
			assert.Equal(t, secCfg.XFrameOptions(), rec.Header().Get(echo.HeaderXFrameOptions))
			assert.Equal(t, secCfg.ContentTypeNosniff(), rec.Header().Get(echo.HeaderXContentTypeOptions))
		})
	})
}
