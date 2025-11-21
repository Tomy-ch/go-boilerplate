//go:generate oapi-codegen --include-tags=health --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=health --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package health は、サーバーのヘルスチェックのハンドラーを提供します。
package health

import (
	"context"

	"boilerplate-go/internal/controller/handler/health/gen"

	"github.com/labstack/echo/v4"
)

type server struct{}

func BindHandler(e *echo.Echo) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{}, nil))
}

// GetHealth は、サーバーのヘルスチェックを行います。
func (s *server) GetHealth(_ context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200JSONResponse(gen.ResponseHealth{
		Status: "ok",
	}), nil
}
