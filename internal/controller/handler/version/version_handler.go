//go:generate oapi-codegen --include-tags=version --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=version --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package version パッケージは、アプリケーションのバージョン情報を提供します。
package version

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/handler/version/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/system"
	"go-boilerplate/pkg/datetime"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

var errInvalidBuildDate = xerrors.Wrap(apperror.ErrInternal, "invalid build date")

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

	buildDate, err := datetime.ParseRFC3339UTCToLocation(s.buildInfo.BuildDate(), s.loc)
	if err != nil {
		return nil, xerrors.Wrap(errInvalidBuildDate, err.Error())
	}

	return gen.GetVersion200JSONResponse(gen.VersionResponse{
		Version:     s.buildInfo.Version(),
		Revision:    s.buildInfo.Revision(),
		BuildDate:   buildDate,
		Environment: s.appCfg.Env(),
		Service:     s.appCfg.Name(),
	}), nil
}
