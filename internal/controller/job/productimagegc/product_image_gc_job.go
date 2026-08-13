// Package productimagegc は、どの商品からも参照されていない商品画像オブジェクトを回収する GC ジョブを提供します。
// 外部スケジューラ（k8s CronJob / cron）が `cmd job product-image-gc` をワンショット起動する想定です。
package productimagegc

import (
	"context"
	"strconv"
	"strings"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
	"go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/xerrors"
)

const jobName = "product-image-gc"

const (
	olderThanFlagPrefix = "--older-than="
	batchSizeFlagPrefix = "--batch-size="
	dryRunFlag          = "--dry-run"
)

var (
	// errUnknownFlag は、未知のフラグが指定された場合のエラーです。
	errUnknownFlag = xerrors.New("unknown flag")
	// errDuplicateFlag は、同一フラグが複数回指定された場合のエラーです。
	errDuplicateFlag = xerrors.New("duplicate flag")
	// errInvalidOlderThan は、--older-than に正の duration 以外が指定された場合のエラーです。
	errInvalidOlderThan = xerrors.New("invalid older-than")
	// errInvalidBatchSize は、--batch-size に正の整数以外が指定された場合のエラーです。
	errInvalidBatchSize = xerrors.New("invalid batch size")
)

// options は、args から読み取った実行条件です。grace / batchSize の 0 は「未指定」を表し、
// usecase 側の既定値が使われます。
type options struct {
	grace     time.Duration
	batchSize int32
	dryRun    bool
}

type jobImpl struct {
	logging logging.Logger
	tracer  observability.LayerTracer
	gc      product.ImageGCUsecase
}

// New は、商品画像 GC ジョブを初期化します。
func New(
	logging logging.Logger,
	tf observability.TracerFactory,
	gc product.ImageGCUsecase,
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

// Execute は、猶予期間を過ぎた未参照の商品画像オブジェクトを回収し、件数をログに出力します。
// --older-than=<duration> で猶予期間を、--batch-size=N で 1 ページの列挙件数を指定でき、
// --dry-run では削除せず対象件数だけを出力します。
// 途中で失敗した場合も、確定した削除件数を出力してからエラーを返します。
func (j *jobImpl) Execute(ctx context.Context, args []string) error {
	ctx, endSpan := j.tracer.Start(ctx)
	defer endSpan()

	opts, err := parseArgs(args)
	if err != nil {
		return err
	}

	result, err := j.gc.SweepOrphans(ctx, opts.grace, opts.batchSize, opts.dryRun)
	fields := []*logging.Field{
		logging.Int64(logging.JobResultKey, result.Deleted),
		logging.Int64(logging.JobScannedKey, result.Scanned),
	}
	if err != nil {
		j.logging.Named(jobName).Warn(ctx, abortedMessage(opts.dryRun), fields...)
		return err
	}

	j.logging.Named(jobName).Info(ctx, resultMessage(opts.dryRun), fields...)
	return nil
}

// parseArgs は、--older-than=<duration> / --batch-size=N / --dry-run を解釈します。
// 未指定の grace / batchSize は 0（usecase 側で既定値）です。
// 同一フラグの重複・未知フラグは、後勝ちで黙殺せずエラーにします。
func parseArgs(args []string) (options, error) {
	var (
		opts          options
		seenOlderThan bool
		seenBatchSize bool
	)
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, olderThanFlagPrefix):
			if seenOlderThan {
				return options{}, xerrors.Wrap(errDuplicateFlag, olderThanFlagPrefix)
			}
			seenOlderThan = true

			d, err := time.ParseDuration(strings.TrimPrefix(a, olderThanFlagPrefix))
			if err != nil || d <= 0 {
				return options{}, xerrors.Wrap(errInvalidOlderThan, a)
			}
			opts.grace = d
		case strings.HasPrefix(a, batchSizeFlagPrefix):
			if seenBatchSize {
				return options{}, xerrors.Wrap(errDuplicateFlag, batchSizeFlagPrefix)
			}
			seenBatchSize = true

			n, err := strconv.ParseInt(strings.TrimPrefix(a, batchSizeFlagPrefix), 10, 32)
			if err != nil || n <= 0 {
				return options{}, xerrors.Wrap(errInvalidBatchSize, a)
			}
			opts.batchSize = int32(n)
		case a == dryRunFlag:
			if opts.dryRun {
				return options{}, xerrors.Wrap(errDuplicateFlag, dryRunFlag)
			}
			opts.dryRun = true
		default:
			return options{}, xerrors.Wrap(errUnknownFlag, a)
		}
	}
	return opts, nil
}

// resultMessage は、完走時の結果ログのメッセージを返します。dry-run では削除していないことを明示します。
func resultMessage(dryRun bool) string {
	if dryRun {
		return "Result: orphaned product images found (dry-run, nothing deleted)"
	}
	return "Result: orphaned product images deleted"
}

// abortedMessage は、中断時の結果ログのメッセージを返します。
// dry-run では削除していないことを、実削除では併記した件数が既に削除済みであることを明示します。
func abortedMessage(dryRun bool) string {
	if dryRun {
		return "Result: product image GC aborted (dry-run, nothing deleted)"
	}
	return "Result: product image GC aborted, the reported images are already deleted"
}
