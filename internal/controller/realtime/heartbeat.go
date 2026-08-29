package realtime

import (
	"context"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

const heartbeatLoggerName = "realtime-heartbeat"

// Heartbeat は、instance lease を LeaseHeartbeatInterval ごとに書き直す常駐 loop です。
// 1 回の失敗では止まらず次の周期で書き直します（失敗が LeaseExpiry を超えて続けば orphan として回収される）。
type Heartbeat struct {
	keeper  ucrealtime.LeaseKeeper
	id      rt.InstanceID
	sleeper clock.Sleeper
	logging logging.Logger
	tracer  observability.LayerTracer
}

// NewHeartbeat は、id の instance の heartbeat loop を生成します。
func NewHeartbeat(
	keeper ucrealtime.LeaseKeeper,
	id rt.InstanceID,
	sleeper clock.Sleeper,
	log logging.Logger,
	tf observability.TracerFactory,
) *Heartbeat {
	return &Heartbeat{keeper: keeper, id: id, sleeper: sleeper, logging: log, tracer: tf.Controller()}
}

// Run は、最初の 1 回を即座に書き、以後 LeaseHeartbeatInterval ごとに書き直します。ctx 完了で nil を返します。
func (h *Heartbeat) Run(ctx context.Context) error {
	ctx, endSpan := h.tracer.Start(ctx)
	defer endSpan()

	log := h.logging.Named(heartbeatLoggerName)
	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := h.keeper.Beat(ctx, h.id); err != nil {
			if ctx.Err() != nil {
				return nil
			}

			log.Error(ctx, "failed to heartbeat instance lease", logging.Error(logging.ErrorKey, err))
		}

		if h.sleeper.Sleep(ctx, ucrealtime.LeaseHeartbeatInterval) != nil {
			return nil
		}
	}
}
