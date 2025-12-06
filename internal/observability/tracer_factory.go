package observability

import (
	"boilerplate-go/internal/logging"
	"boilerplate-go/pkg/fnmeta"

	"go.opentelemetry.io/otel/trace"
)

type TracerFactory interface {
	// Controller は、コントローラー層用のトレーサーを返します。pkg はパッケージ名。
	Controller() LayerTracer
	// Usecase は、ユースケース層用のトレーサーを返します。
	Usecase() LayerTracer
	// Infra は、インフラ層用のトレーサーを返します。
	Infra() LayerTracer
}

type tracerFactory struct {
	tp  trace.TracerProvider
	log logging.Logger
	lf  logging.LogFieldBuilder
}

// NewTracerFactory は、TracerFactory を初期化して返します。
func NewTracerFactory(tp trace.TracerProvider, log logging.Logger, lf logging.LogFieldBuilder) TracerFactory {
	return &tracerFactory{
		tp:  tp,
		log: log,
		lf:  lf,
	}
}

// Controller は Controller 層用のトレーサーを返します。
func (t *tracerFactory) Controller() LayerTracer {
	full := getCallerFullName()
	pkg := fnmeta.ExtractPackageName(full)

	return LayerTracer{
		log:     t.log,
		lf:      t.lf,
		tracer:  t.tp.Tracer(tracerNameController + delimiter + pkg),
		layer:   tracerNameController,
		pkgName: pkg,
	}
}

// Usecase は Usecase 層用のトレーサーを返します。
func (t *tracerFactory) Usecase() LayerTracer {
	full := getCallerFullName()
	pkg := fnmeta.ExtractPackageName(full)

	return LayerTracer{
		log:     t.log,
		lf:      t.lf,
		tracer:  t.tp.Tracer(tracerNameUsecase + delimiter + pkg),
		layer:   tracerNameUsecase,
		pkgName: pkg,
	}
}

// Infra は Infrastructure 層用のトレーサーを返します。
func (t *tracerFactory) Infra() LayerTracer {
	full := getCallerFullName()
	pkg := fnmeta.ExtractPackageName(full)

	return LayerTracer{
		log:     t.log,
		lf:      t.lf,
		tracer:  t.tp.Tracer(tracerNameInfrastructure + delimiter + pkg),
		layer:   tracerNameInfrastructure,
		pkgName: pkg,
	}
}
