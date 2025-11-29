//go:generate oapi-codegen --include-tags=healthz --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=healthz --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package healthz は、サーバーのヘルスチェックのハンドラーを提供します。
package healthz

import (
	"context"

	"boilerplate-go/internal/controller/handler/healthz/gen"
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

// GetHealthz は、サーバーのヘルスチェックを行います。
func (s *server) GetHealthz(
	ctx context.Context, _ gen.GetHealthzRequestObject,
) (gen.GetHealthzResponseObject, error) {
	_, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	return gen.GetHealthz200JSONResponse(gen.ResponseHealth{
		Status: "ok",
	}), nil
}
