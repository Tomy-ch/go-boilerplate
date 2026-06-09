// Package usercount は、ユーザの件数をカウントするジョブを提供するパッケージです。
package usercount

import (
	"context"
	"fmt"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/pkg/ptr"
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

// Execute は、ジョブを実行します。
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
				return xerrors.New(fmt.Sprintf("conflicting filter flag: %s", a))
			}
			active = ptr.To(true)
			filter = "active"
		case "--inactive-only":
			if active != nil {
				return xerrors.New(fmt.Sprintf("conflicting filter flag: %s", a))
			}
			active = ptr.To(false)
			filter = "inactive"
		default:
			return xerrors.New(fmt.Sprintf("unknown flag: %s", a))
		}
	}
	count, err := u.usecase.CountUsers(ctx, active)
	if err != nil {
		return err
	}
	u.logging.Named(jobName).Info(
		"Result: total user count",
		logging.Int64(logging.JobResultKey, count),
		logging.String("filter", filter),
	)
	return nil
}
