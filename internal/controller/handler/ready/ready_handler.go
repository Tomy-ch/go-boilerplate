//go:generate oapi-codegen --include-tags=ready --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=ready --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package ready は、サーバーのレディネスチェックのハンドラーを提供します。
package ready

import (
	"context"

	"boilerplate-go/internal/controller/handler/ready/gen"
	healthcheckuc "boilerplate-go/internal/usecase/healthcheck"

	"github.com/labstack/echo/v4"
)

type server struct {
	healthUsecase healthcheckuc.Usecase
}

func BindHandler(e *echo.Echo, healthUsecase healthcheckuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		healthUsecase: healthUsecase,
	}, nil))
}

// GetReady は、サーバーのレディネスチェックを行います。
func (s *server) GetReady(
	ctx context.Context, _ gen.GetReadyRequestObject,
) (gen.GetReadyResponseObject, error) {
	res, err := s.healthUsecase.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetReady200JSONResponse(gen.ResponseReady{
		Status:          gen.ResponseReadyStatus(res.Status),
		ApplicationTime: res.ApplicationTime,
		DbLatencyMs:     res.DBHealthCheck.Latency.Milliseconds(),
		DbResponsedAt:   res.DBHealthCheck.ResponsedAt,
	}), nil
}
