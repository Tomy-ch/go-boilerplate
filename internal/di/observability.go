package di

import (
	"boilerplate-go/internal/controller/httpstack/observability"

	"go.uber.org/fx"
)

func ObservabilityModule() fx.Option {
	return fx.Module("observability",
		observability.PrimitiveModule(),
		observability.CoreModule(),
	)
}
