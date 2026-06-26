// Package worker は、worker 関連の依存性注入を提供します。
package worker

import (
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	workerengine "go-boilerplate/internal/controller/worker"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
)

// EngineIn は、Engine の入力パラメータを表します。
type EngineIn struct {
	fx.In

	Workers []workerboundary.Worker `group:"workers"`
	Config  *config.WorkerConfig
	TF      observability.TracerFactory
	Metrics *observability.WorkerMetrics
	Logger  logging.Logger
}

// ProvideEngine は、WorkerConfig の設定から Engine を生成します。
func ProvideEngine(in EngineIn) (*workerengine.Engine, error) {
	c := in.Config
	set := workerengine.Settings{
		Concurrency:               c.Concurrency(),
		MaxInFlight:               c.MaxInFlight(),
		BatchSize:                 c.BatchSize(),
		ExtendInterval:            c.ExtendInterval(),
		DrainTimeout:              c.DrainTimeout(),
		ReceiveCountWarnThreshold: c.ReceiveCountWarnThreshold(),
		CircuitFailureThreshold:   c.CircuitFailureThreshold(),
		CircuitOpenBackoffInitial: c.CircuitOpenBackoffInitial(),
		CircuitOpenBackoffMax:     c.CircuitOpenBackoffMax(),
		CircuitHalfOpenProbe:      c.CircuitHalfOpenProbe(),
		ProgressStaleAfter:        c.ProgressStaleAfter(),
		NackBackoffInitial:        c.NackBackoffInitial(),
		NackBackoffMax:            c.NackBackoffMax(),
	}
	return workerengine.New(in.Workers, set, in.TF, in.Metrics, in.Logger)
}
