package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/config"
	mock_ratelimit "boilerplate-go/internal/controller/httpstack/ratelimit/mock"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func Test_Middleware(t *testing.T) {
	t.Parallel()

	t.Run("IPRateLimiterのEnabledがfalseの場合、次のハンドラがそのまま呼ばれる", func(t *testing.T) {
		t.Parallel()

		// disabled config (zero value -> Enabled() == false)
		ipCfg := &config.IPRateLimitConfig{}

		mw := Middleware(nil, ipCfg)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		called := false
		h := mw(func(c echo.Context) error {
			called = true
			return c.String(http.StatusOK, "ok")
		})

		err := h(c)
		require.NoError(t, err)
		require.True(t, called)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("IPRateLimiterのEnabledがtrueの場合、次のハンドラがそのまま呼ばれる", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		cfg := config.MockConfigForTest(t)
		ipCfg := config.NewIPRateLimitConfig(cfg)

		t.Run("発行されたURIが運用系APIの場合、次のハンドラがそのまま呼ばれる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			mockRL := mock_ratelimit.NewMockIPRateLimiter(ctrl)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := Middleware(mockRL, ipCfg)
			h := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "allowed")
			})
			err := h(c)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)
		})

		t.Run("許可されるケースの場合、次のハンドラが呼ばれる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			mockRL := mock_ratelimit.NewMockIPRateLimiter(ctrl)
			mockRL.EXPECT().AllowRequest(gomock.Any()).Return(true)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := Middleware(mockRL, ipCfg)
			h := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "allowed")
			})
			err := h(c)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)
		})

		t.Run("拒否されるケースの場合、429が返される", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			mockRL := mock_ratelimit.NewMockIPRateLimiter(ctrl)
			mockRL.EXPECT().AllowRequest(gomock.Any()).Return(false)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := Middleware(mockRL, ipCfg)
			h := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "should not be called")
			})
			err := h(c)
			require.ErrorIs(t, err, apperror.ErrTooManyRequests)
			require.Equal(t, "1", rec.Header().Get("Retry-After"))
		})
	})
}
