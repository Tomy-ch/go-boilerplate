// Package usercount は、ユーザの件数をカウントするジョブを提供するパッケージです。
package usercount

import (
	"context"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/pkg/xerrors"
)

const jobName = "user-count"

type jobImpl struct {
	logging logging.Logger
	tracer  observability.LayerTracer
	usecase user.Usecase
}

// New は、ユーザの件数をカウントするジョブを初期化します。
func New(
	logging logging.Logger,
	tf observability.TracerFactory,
	usecase user.Usecase,
) job.Job {
	return &jobImpl{
		logging: logging,
		tracer:  tf.Controller(),
		usecase: usecase,
	}
}

// Name は、このジョブの名前を返します。
func (u *jobImpl) Name() string {
	return jobName
}

// Execute は、ユーザ件数を集計してログに出力します。
// --active-only / --inactive-only でカウント対象を絞り込めます。両フラグの併用はエラーです。
func (u *jobImpl) Execute(ctx context.Context, args []string) error {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	var active *bool
	filter := "all"
	for _, a := range args {
		// 相反・重複するフィルタ指定は、後勝ちで黙殺せずエラーにする。
		switch a {
		case "--active-only":
			if active != nil {
				return xerrors.New("conflicting filter flag: " + a)
			}
			active = new(true)
			filter = "active"
		case "--inactive-only":
			if active != nil {
				return xerrors.New("conflicting filter flag: " + a)
			}
			active = new(false)
			filter = "inactive"
		default:
			return xerrors.New("unknown flag: " + a)
		}
	}
	count, err := u.usecase.CountUsers(ctx, active)
	if err != nil {
		return err
	}
	u.logging.Named(jobName).Info(
		ctx,
		"Result: total user count",
		logging.Int64(logging.JobResultKey, count),
		logging.String(logging.FilterKey, filter),
	)
	return nil
}
