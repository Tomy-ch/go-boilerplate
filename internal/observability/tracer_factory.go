//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package observability

import (
	"go-boilerplate/pkg/fnmeta"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerFactory は、レイヤー別の LayerTracer を生成するファクトリです。
type TracerFactory interface {
	// Controller は、コントローラー層用のトレーサーを返します。
	Controller() LayerTracer
	// Usecase は、ユースケース層用のトレーサーを返します。
	Usecase() LayerTracer
	// Infra は、インフラ層用のトレーサーを返します。
	Infra() LayerTracer
}

type tracerFactory struct {
	tp trace.TracerProvider
}

// NewTracerFactory は、TracerFactory を初期化して返します。
func NewTracerFactory(tp trace.TracerProvider) TracerFactory {
	return &tracerFactory{
		tp: tp,
	}
}

// NewDisabledTracerFactory は、スパンを一切送出しない TracerFactory を返します。
// DI グラフを組まない CLI が infra 実装を直接組み立てる場合に用います
// （otel パッケージはこの層の外では使えないため、生成をここへ閉じます）。
func NewDisabledTracerFactory() TracerFactory {
	return NewTracerFactory(noop.NewTracerProvider())
}

// Controller/Usecase/Infra は本体をインライン重複させている。getCallerFullName のスキップ段数(2)が
// 呼び出し階層に依存するため、共通ヘルパへ括り出すと pkg 解決が 1 段ずれて壊れる。

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
		tracer:  t.tp.Tracer(string(layer) + delimiter + pkgName),
		layer:   layer,
		pkgName: pkgName,
	}
}
