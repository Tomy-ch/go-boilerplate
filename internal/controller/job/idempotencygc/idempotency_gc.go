// Package idempotencygc は、TTL 失効した冪等性キーを削除する GC ジョブを提供します。
// 外部スケジューラ（k8s CronJob / cron）が `cmd job idempotency-gc` をワンショット起動する想定です。
package idempotencygc

import (
	"context"
	"strconv"
	"strings"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/pkg/xerrors"
)

const jobName = "idempotency-gc"

const batchSizeFlagPrefix = "--batch-size="

const (
	// resultMessage は、完走時の結果ログのメッセージです。
	resultMessage = "Result: expired idempotency keys deleted"
	// abortedMessage は、中断時の結果ログのメッセージです。
	abortedMessage = "Result: idempotency GC aborted, the reported keys are already deleted"
)

var (
	// errUnknownFlag は、未知のフラグが指定された場合のエラーです。
	errUnknownFlag = xerrors.New("unknown flag")
	// errDuplicateFlag は、同一フラグが複数回指定された場合のエラーです。
	errDuplicateFlag = xerrors.New("duplicate flag")
	// errInvalidBatchSize は、--batch-size に正の整数以外が指定された場合のエラーです。
	errInvalidBatchSize = xerrors.New("invalid batch size")
)

type jobImpl struct {
	logging logging.Logger
	tracer  observability.LayerTracer
	gc      idempotency.GCUsecase
}

// New は、冪等性キー GC ジョブを初期化します。
func New(
	logging logging.Logger,
	tf observability.TracerFactory,
	gc idempotency.GCUsecase,
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

// Execute は、失効した冪等性キーをバッチ削除します。--batch-size=N で 1 バッチの件数を指定できます。
// 途中で失敗した場合も、確定した削除件数を出力してからエラーを返します。
func (j *jobImpl) Execute(ctx context.Context, args []string) error {
	ctx, endSpan := j.tracer.Start(ctx)
	defer endSpan()

	batchSize, err := parseBatchSize(args)
	if err != nil {
		return err
	}

	deleted, err := j.gc.SweepExpired(ctx, batchSize)
	if err != nil {
		// 中断までにコミットされた削除は取り消せない。エラーだけを返すと消えた件数が運用者に届かないため、
		// 確定した件数を記録してから伝播する。
		j.logging.Named(jobName).Warn(ctx, abortedMessage, logging.Int64(logging.JobResultKey, deleted))
		return err
	}

	j.logging.Named(jobName).Info(ctx, resultMessage, logging.Int64(logging.JobResultKey, deleted))
	return nil
}

// parseBatchSize は、--batch-size=N フラグを解釈します。未指定なら 0（usecase 側で既定値）。
func parseBatchSize(args []string) (int32, error) {
	var batchSize int32
	seen := false
	for _, a := range args {
		if !strings.HasPrefix(a, batchSizeFlagPrefix) {
			return 0, xerrors.Wrap(errUnknownFlag, a)
		}
		if seen {
			return 0, xerrors.Wrap(errDuplicateFlag, batchSizeFlagPrefix)
		}
		seen = true

		n, err := strconv.ParseInt(strings.TrimPrefix(a, batchSizeFlagPrefix), 10, 32)
		if err != nil || n <= 0 {
			return 0, xerrors.Wrap(errInvalidBatchSize, a)
		}
		batchSize = int32(n)
	}
	return batchSize, nil
}
