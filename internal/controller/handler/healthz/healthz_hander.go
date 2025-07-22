//go:generate oapi-codegen --include-tags=healthz --package=gen --generate=types -o ./gen/type.gen.go $PJ_DIR/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=healthz --package=gen --generate=echo-server -o ./gen/server.gen.go $PJ_DIR/openapi/openapi.gen.yaml
package healthz

import (
	"net/http"

	"boilerplate-go/internal/controller/handler/healthz/gen"

	"github.com/labstack/echo/v4"
)

type server struct{}

func BindHandler(e *echo.Echo) {
	gen.RegisterHandlers(e, &server{})
}

// GetHealthz は、サーバーのヘルスチェックを行います。
func (s *server) GetHealthz(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, &gen.ResponseHealth{
		Status: "ok",
	})
}
