// Package outboxgc は、retention を超えた published 済みの outbox 行を削除する GC ジョブを提供します。
// 外部スケジューラ（k8s CronJob / cron）が `cmd job outbox-gc` をワンショット起動する想定です。
package outboxgc

import (
	"context"
	"strconv"
	"strings"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
	"go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/xerrors"
)

const jobName = "outbox-gc"

const batchSizeFlagPrefix = "--batch-size="

type jobImpl struct {
	logging logging.Logger
	tracer  observability.LayerTracer
	gc      outbox.GCUsecase
}

// New は、outbox GC ジョブを初期化します。
func New(
	logging logging.Logger,
	tf observability.TracerFactory,
	gc outbox.GCUsecase,
) job.Job {
	return &jobImpl{
		logging: logging,
		tracer:  tf.Controller(),
		gc:      gc,
	}
}

// Name は、このジョブの名前を返します。
func (j *jobImpl) Name() string {
	return jobName
}

// Execute は、retention を超えた published 行をバッチ削除します。--batch-size=N で 1 バッチの件数を指定できます。
func (j *jobImpl) Execute(ctx context.Context, args []string) error {
	ctx, endSpan := j.tracer.Start(ctx)
	defer endSpan()

	batchSize, err := parseBatchSize(args)
	if err != nil {
		return err
	}

	deleted, err := j.gc.SweepPublished(ctx, batchSize)
	if err != nil {
		return err
	}

	j.logging.Named(jobName).Info(
		ctx,
		"Result: published outbox rows deleted",
		logging.Int64(logging.JobResultKey, deleted),
	)
	return nil
}

// parseBatchSize は、--batch-size=N フラグを解釈します。未指定なら 0（usecase 側で既定値）。
func parseBatchSize(args []string) (int32, error) {
	var batchSize int32
	seen := false
	for _, a := range args {
		if !strings.HasPrefix(a, batchSizeFlagPrefix) {
			return 0, xerrors.New("unknown flag: " + a)
		}
		if seen {
			return 0, xerrors.New("duplicate flag: " + batchSizeFlagPrefix)
		}
		seen = true

		n, err := strconv.ParseInt(strings.TrimPrefix(a, batchSizeFlagPrefix), 10, 32)
		if err != nil || n <= 0 {
			return 0, xerrors.New("invalid batch size: " + a)
		}
		batchSize = int32(n)
	}
	return batchSize, nil
}
