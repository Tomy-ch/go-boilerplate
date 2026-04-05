// Package usercount は、ユーザの件数をカウントするジョブを提供するパッケージです。
package usercount

import (
	"context"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/pkg/ptr"
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
	for _, a := range args {
		switch a {
		case "--active-only":
			active = ptr.To(true)
		case "--inactive-only":
			active = ptr.To(false)
		}
	}
	count, err := u.usecase.CountUsers(ctx, active)
	if err != nil {
		return err
	}
	u.logging.Named(jobName).Info(
		"Result: total user count",
		logging.Int64(logging.JobResultKey, count),
	)
	return nil
}
