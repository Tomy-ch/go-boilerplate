// Package orphancleanup は、死んだ serve instance が残した Realtime の受信先（queue と subscription）と
// その lease を回収する one-shot ジョブを提供します。外部スケジューラ（k8s CronJob / cron）が
// `cmd job orphan-cleanup` をワンショット起動する想定で、アプリケーション内では起動しません（ADR-0109）。
package orphancleanup

import (
	"context"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/xerrors"
)

const jobName = "orphan-cleanup"

const (
	// resultMessage は、完走時の結果ログのメッセージです。
	resultMessage = "Result: orphaned realtime instances reclaimed"
	// abortedMessage は、一部が失敗したときの結果ログのメッセージです。
	abortedMessage = "Result: orphan cleanup incomplete, the reported counts are already applied"
)

// errUnknownFlag は、未知のフラグが指定された場合のエラーです。
var errUnknownFlag = xerrors.New("unknown flag")

// SweeperFactory は、掃除役を実行時に組み立てる関数です。Realtime の依存は DI graph に載せず、
// このジョブが選ばれたときにだけ構築します（理由は README「Public API」の SweeperFactory）。
type SweeperFactory func(ctx context.Context) (ucrealtime.OrphanSweeper, error)

type jobImpl struct {
	logging    logging.Logger
	tracer     observability.LayerTracer
	newSweeper SweeperFactory
}

// New は、orphan cleanup ジョブを初期化します。
func New(
	logging logging.Logger,
	tf observability.TracerFactory,
	newSweeper SweeperFactory,
) job.Job {
	return &jobImpl{
		logging:    logging,
		tracer:     tf.Controller(),
		newSweeper: newSweeper,
	}
}

// Name は、このジョブの名前を返します。
func (j *jobImpl) Name() string {
	return jobName
}

// Execute は、回収できる instance をすべて処理します。引数は取りません。
// 一部が失敗した場合も、確定した内訳を出力してからエラーを返します。個々の失敗はそのエラーの chain に
// 残るため、内訳には失敗の件数を載せません。
func (j *jobImpl) Execute(ctx context.Context, args []string) error {
	ctx, endSpan := j.tracer.Start(ctx)
	defer endSpan()

	if len(args) > 0 {
		return xerrors.Wrap(errUnknownFlag, args[0])
	}

	sweeper, err := j.newSweeper(ctx)
	if err != nil {
		return err
	}

	result, err := sweeper.Sweep(ctx)
	if err != nil {
		j.logging.Named(jobName).Warn(ctx, abortedMessage, resultFields(result)...)

		return err
	}

	j.logging.Named(jobName).Info(ctx, resultMessage, resultFields(result)...)

	return nil
}

// resultFields は、掃除の内訳をジョブ共通のログフィールドへ写します。
func resultFields(r ucrealtime.SweepResult) []*logging.Field {
	return []*logging.Field{
		logging.Int(logging.JobScannedKey, r.Detected),
		logging.Int(logging.JobResultKey, r.Reclaimed),
		logging.Int(logging.JobSkippedKey, r.Skipped),
	}
}
