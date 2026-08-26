// Package userpurge は、退会から retention を過ぎたユーザーを物理削除するジョブを提供します。
// 外部スケジューラ（k8s CronJob / cron）が `cmd job user-purge` をワンショット起動する想定です。
package userpurge

import (
	"context"
	"strconv"
	"strings"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/pkg/xerrors"
)

const jobName = "user-purge"

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

// options は、args から読み取った実行条件です。retention / batchSize の 0 は「未指定」を表し、
// usecase 側の既定値が使われます。
type options struct {
	retention time.Duration
	batchSize int32
	dryRun    bool
}

type jobImpl struct {
	logging logging.Logger
	tracer  observability.LayerTracer
	purge   user.PurgeUsecase
}

// New は、ユーザー物理削除ジョブを初期化します。
func New(
	logging logging.Logger,
	tf observability.TracerFactory,
	purge user.PurgeUsecase,
) job.Job {
	return &jobImpl{
		logging: logging,
		tracer:  tf.Controller(),
		purge:   purge,
	}
}

// Name は、このジョブの名前を返します。
func (j *jobImpl) Name() string {
	return jobName
}

// Execute は、退会から retention を過ぎたユーザーをバッチ物理削除し、件数をログに出力します。
// --older-than=<duration> で保持期間を、--batch-size=N で 1 バッチの件数を指定でき、
// --dry-run では削除せず対象件数だけを出力します。
// 途中で失敗した場合も、確定した件数を出力してからエラーを返します。
func (j *jobImpl) Execute(ctx context.Context, args []string) error {
	ctx, endSpan := j.tracer.Start(ctx)
	defer endSpan()

	opts, err := parseArgs(args)
	if err != nil {
		return err
	}

	result, err := j.purge.PurgeDeleted(ctx, opts.retention, opts.batchSize, opts.dryRun)
	fields := []*logging.Field{
		logging.Int64(logging.JobResultKey, result.Purged),
		logging.Int64(logging.JobSkippedKey, result.SkippedWithPurchases),
	}
	if err != nil {
		j.logging.Named(jobName).Warn(ctx, abortedMessage(opts.dryRun), fields...)
		return err
	}

	j.logging.Named(jobName).Info(ctx, resultMessage(opts.dryRun), fields...)
	return nil
}

// parseArgs は、--older-than=<duration> / --batch-size=N / --dry-run を解釈します。
// 未指定の retention / batchSize は 0（usecase 側で既定値）です。
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
			opts.retention = d
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
		return "Result: purge target users (dry-run, nothing deleted)"
	}
	return "Result: deleted users purged"
}

// abortedMessage は、中断時の結果ログのメッセージを返します。
// dry-run では削除していないことを、実削除では併記した件数が既にコミット済みであることを明示します。
func abortedMessage(dryRun bool) string {
	if dryRun {
		return "Result: purge aborted (dry-run, nothing deleted)"
	}
	return "Result: purge aborted, the reported users are already deleted"
}
