//go:generate oapi-codegen --include-tags=health --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=health --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package health は、サーバーのヘルスチェックのハンドラーを提供します。
package health

import (
	"context"

	"boilerplate-go/internal/controller/handler/health/gen"
	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
)

type server struct {
	tracer observability.LayerTracer
}

func BindHandler(
	e *echo.Echo, tf observability.TracerFactory,
) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
	}, nil))
}

// GetHealth は、サーバーのヘルスチェックを行います。
func (s *server) GetHealth(ctx context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	_, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	// time.Sleep(10 * time.Second)

	return gen.GetHealth200JSONResponse(gen.ResponseHealth{
		Status: "ok",
	}), nil
}
