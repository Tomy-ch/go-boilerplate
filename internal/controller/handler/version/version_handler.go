//go:generate oapi-codegen --include-tags=version --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=version --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package version パッケージは、アプリケーションのバージョン情報を提供します。
package version

import (
	"context"
	"time"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler/version/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/system"
	"boilerplate-go/pkg/datetime"
	"boilerplate-go/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

type server struct {
	buildInfo system.BuildInfo
	appCfg    *config.ApplicationConfig
	loc       *time.Location
	tracer    observability.LayerTracer
}

func BindHandler(
	e *echo.Echo,
	tf observability.TracerFactory,
	loc *time.Location,
	bi system.BuildInfo,
	ac *config.ApplicationConfig,
) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		buildInfo: bi,
		appCfg:    ac,
		loc:       loc,
		tracer:    tf.Controller(),
	}, nil))
}

// GetVersion は、アプリケーションのバージョン情報を取得します。
func (s *server) GetVersion(ctx context.Context, _ gen.GetVersionRequestObject) (gen.GetVersionResponseObject, error) {
	_, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	buildDate, err := datetime.ParseRFC3339UTCInLocation(s.buildInfo.BuildDate(), s.loc)
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, err.Error())
	}

	return gen.GetVersion200JSONResponse(gen.VersionResponse{
		Version:     s.buildInfo.Version(),
		Revision:    s.buildInfo.Revision(),
		BuildDate:   buildDate,
		Environment: s.appCfg.Env(),
		Service:     s.appCfg.Name(),
	}), nil
}
