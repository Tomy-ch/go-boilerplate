//go:generate oapi-codegen --include-tags=health --package=gen --generate=types -o ./gen/type.gen.go $PJ_DIR/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=health --package=gen --generate=echo-server -o ./gen/server.gen.go $PJ_DIR/openapi/openapi.gen.yaml
package health

import (
	"net/http"

	"boilerplate-go/internal/controller/handler/health/gen"

	"github.com/labstack/echo/v4"
)

type server struct{}

func BindHandler(e *echo.Echo) {
	gen.RegisterHandlers(e, &server{})
}

// GetHealth は、サーバーのヘルスチェックを行います。
func (s *server) GetHealth(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, &gen.ResponseHealth{
		Status: "ok",
	})
}
