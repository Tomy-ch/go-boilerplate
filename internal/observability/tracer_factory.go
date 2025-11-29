package observability

import (
	"boilerplate-go/pkg/fnmeta"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
	tp trace.TracerProvider
	z  *zap.Logger
}

const (
	delimiter                = "."
	tracerNameController     = "controller"
	tracerNameUsecase        = "usecase"
	tracerNameInfrastructure = "infrastructure"
)

// NewTracerFactory は、TracerFactory を初期化して返します。
func NewTracerFactory(tp trace.TracerProvider, z *zap.Logger) TracerFactory {
	return &tracerFactory{
		tp: tp,
		z:  z,
	}
}

// Controller は Controller 層用のトレーサーを返します。
func (t *tracerFactory) Controller() LayerTracer {
	full := getCallerFullName()
	pkg := fnmeta.ExtractPackageName(full)

	return LayerTracer{
		log:     t.z,
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
		log:     t.z,
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
		log:     t.z,
		tracer:  t.tp.Tracer(tracerNameInfrastructure + delimiter + pkg),
		layer:   tracerNameInfrastructure,
		pkgName: pkg,
	}
}
