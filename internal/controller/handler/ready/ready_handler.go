//go:generate oapi-codegen --include-tags=ready --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=ready --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package ready は、サーバーのレディネスチェックのハンドラーを提供します。
package ready

import (
	"context"

	"go-boilerplate/internal/controller/handler/ready/gen"
	"go-boilerplate/internal/observability"
	healthcheckuc "go-boilerplate/internal/usecase/healthcheck"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer        observability.LayerTracer
	healthUsecase healthcheckuc.Usecase
}

// BindHandler は、レディネスチェックのハンドラーをEchoに登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, healthUsecase healthcheckuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer:        tf.Controller(),
		healthUsecase: healthUsecase,
	}, nil))
}

func (s *server) GetReady(
	ctx context.Context, _ gen.GetReadyRequestObject,
) (gen.GetReadyResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	res, err := s.healthUsecase.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetReady200JSONResponse(gen.ReadyResponse{
		Status:          gen.ReadyResponseStatus(res.Status),
		ApplicationTime: res.ApplicationTime,
		DbLatencyMs:     res.DBHealthCheck.Latency.Milliseconds(),
		DbRespondedAt:   res.DBHealthCheck.RespondedAt,
		Dependencies:    toDependencies(res.Dependencies),
	}), nil
}

// toDependencies は、依存の状態をレスポンスの表現へ写します。
// 依存が 1 つも無ければ項目そのものを出しません（結線されていない環境で空配列を見せないため）。
func toDependencies(deps []healthcheckuc.DependencyStatus) *[]gen.ReadyDependency {
	if len(deps) == 0 {
		return nil
	}

	out := make([]gen.ReadyDependency, 0, len(deps))
	for _, d := range deps {
		out = append(out, gen.ReadyDependency{Name: d.Name, Status: gen.ReadyDependencyStatus(d.Status)})
	}

	return &out
}
