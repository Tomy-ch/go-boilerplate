package security

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/require"
)

func TestBuildSecureConfig(t *testing.T) {
	t.Parallel()
	t.Run("本番モード", func(t *testing.T) {
		t.Parallel()
		cfg := config.MockConfigForTest(t)
		cfg.SetServerAppMode(t, config.ProductionMode)

		actual := buildSecureConfig(&cfg)

		expected := middleware.SecureConfig{
			XSSProtection:         "",
			ContentTypeNosniff:    "nosniff",
			XFrameOptions:         "DENY",
			ReferrerPolicy:        "no-referrer",
			HSTSExcludeSubdomains: false,
			HSTSMaxAge:            31536000,
		}

		require.Equal(t, expected, actual)
	})

	t.Run("非本番モード", func(t *testing.T) {
		t.Parallel()
		cfg := config.MockConfigForTest(t)
		cfg.SetServerAppMode(t, config.DevelopmentMode)

		actual := buildSecureConfig(&cfg)

		expected := middleware.SecureConfig{
			XSSProtection:      "",
			ContentTypeNosniff: "nosniff",
			XFrameOptions:      "DENY",
			ReferrerPolicy:     "no-referrer",
		}

		require.Equal(t, expected, actual)
	})
}
