//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

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

	return t.newLayerTracer(Controller, pkg)
}

// Usecase は Usecase 層用のトレーサーを返します。
func (t *tracerFactory) Usecase() LayerTracer {
	full := getCallerFullName()
	pkg := fnmeta.ExtractPackageName(full)

	return t.newLayerTracer(Usecase, pkg)
}

// Infra は Infrastructure 層用のトレーサーを返します。
func (t *tracerFactory) Infra() LayerTracer {
	full := getCallerFullName()
	pkg := fnmeta.ExtractPackageName(full)

	return t.newLayerTracer(Infra, pkg)
}

// newLayerTracer は LayerTracer を初期化して返します。
func (t *tracerFactory) newLayerTracer(layer layerName, pkgName string) LayerTracer {
	return LayerTracer{
		log:     t.log,
		lf:      t.lf,
		tracer:  t.tp.Tracer(string(layer) + delimiter + pkgName),
		layer:   layer,
		pkgName: pkgName,
	}
}
