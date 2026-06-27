// Package worker は、worker 関連の依存性注入を提供します。
package worker

import (
	"fmt"
	"time"

	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	workerengine "go-boilerplate/internal/controller/worker"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/xerrors"
)

// ErrInvalidShutdownGrace は、停止猶予より drain の方が長く設定された不整合を表します。
var ErrInvalidShutdownGrace = xerrors.New("invalid shutdown grace")

// EngineIn は、Engine の入力パラメータを表します。
type EngineIn struct {
	fx.In

	Workers []workerboundary.Worker `group:"workers"`
	Config  *config.WorkerConfig
	TF      observability.TracerFactory
	Metrics *observability.WorkerMetrics
	Logger  logging.Logger
}

// ValidateShutdownGrace は、drain 完了前に停止猶予が尽きないことを起動時に検証します。
//
// 停止時の実効カットオフは fx.StopTimeout と停止 context の deadline（共に APP_SHUTDOWN_TIMEOUT を
// 単一軸とする grace）で決まります。WORKER_DRAIN_TIMEOUT >= grace だと OnStop の drain が grace で
// 打ち切られ、未完了の in-flight handler と後続の停止処理（DB pool close 等）が競合します。
// 設定不整合は起動時に失敗させ、運用時の競合を未然に防ぎます。
func ValidateShutdownGrace(appCfg *config.ApplicationConfig, workerCfg *config.WorkerConfig) error {
	return validateShutdownGrace(workerCfg.DrainTimeout(), appCfg.ShutdownTimeout())
}

// validateShutdownGrace は、drain < grace を検証します。
func validateShutdownGrace(drain, grace time.Duration) error {
	if drain >= grace {
		return xerrors.Wrap(ErrInvalidShutdownGrace, fmt.Sprintf(
			"WORKER_DRAIN_TIMEOUT (%s) must be shorter than APP_SHUTDOWN_TIMEOUT (%s)", drain, grace))
	}
	return nil
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
