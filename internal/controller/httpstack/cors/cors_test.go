package cors

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/require"
)

func Test_buildCORSConfig(t *testing.T) {
	t.Parallel()

	t.Run("正常系/引数のallowedOriginsに応じてCORS設定が正しく構築されること", func(t *testing.T) {
		t.Parallel()

		allowedOrigins := []string{"http://localhost:3000", "http://localhost:4000"}
		expected := middleware.CORSConfig{
			AllowOrigins: allowedOrigins,
			AllowMethods: []string{
				echo.HEAD,
				echo.GET,
				echo.POST,
				echo.PUT,
				echo.PATCH,
				echo.DELETE,
				echo.OPTIONS,
			},
			AllowHeaders: []string{
				echo.HeaderOrigin,
				echo.HeaderContentType,
				echo.HeaderAccept,
				echo.HeaderAuthorization,
			},
			ExposeHeaders: []string{
				echo.HeaderContentDisposition,
				echo.HeaderLocation,
				echo.HeaderXRequestID,
			},
			AllowCredentials: false,
		}

		actual := buildCORSConfig(allowedOrigins)

		require.Equal(t, expected, actual)
	})
}
