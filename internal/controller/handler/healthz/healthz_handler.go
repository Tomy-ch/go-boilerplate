//go:generate oapi-codegen --include-tags=healthz --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=healthz --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package healthz は、サーバーのヘルスチェックのハンドラーを提供します。
package healthz

import (
	"context"

	"boilerplate-go/internal/controller/handler/healthz/gen"

	"github.com/labstack/echo/v4"
)

type server struct{}

func BindHandler(e *echo.Echo) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{}, nil))
}

// GetHealthz は、サーバーのヘルスチェックを行います。
func (s *server) GetHealthz(
	_ context.Context, _ gen.GetHealthzRequestObject,
) (gen.GetHealthzResponseObject, error) {
	return gen.GetHealthz200JSONResponse(gen.ResponseHealth{
		Status: "ok",
	}), nil
}
