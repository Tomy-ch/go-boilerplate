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

const (
	activeOnlyFlag   = "--active-only"
	inactiveOnlyFlag = "--inactive-only"
)

var (
	// errDuplicateFlag は、同一フラグが複数回指定された場合のエラーです。
	errDuplicateFlag = xerrors.New("duplicate flag")
	// errUnknownFlag は、未知のフラグが指定された場合のエラーです。
	errUnknownFlag = xerrors.New("unknown flag")
	// errConflictingFilterFlags は、--active-only と --inactive-only が併用された場合のエラーです。
	errConflictingFilterFlags = xerrors.New("conflicting filter flags")
)

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
func (j *jobImpl) Name() string {
	return jobName
}

// Execute は、ユーザ件数を集計してログに出力します。
// --active-only / --inactive-only でカウント対象を絞り込めます。両フラグの併用はエラーです。
func (j *jobImpl) Execute(ctx context.Context, args []string) error {
	ctx, endSpan := j.tracer.Start(ctx)
	defer endSpan()

	active, err := parseFilter(args)
	if err != nil {
		return err
	}

	count, err := j.usecase.CountUsers(ctx, active)
	if err != nil {
		return err
	}
	j.logging.Named(jobName).Info(
		ctx,
		"Result: total user count",
		logging.Int64(logging.JobResultKey, count),
		logging.String(logging.FilterKey, filterLabel(active)),
	)
	return nil
}

// parseFilter は、--active-only / --inactive-only フラグを解釈しカウント対象の絞り込み条件を返します。
// 未指定は nil（全件）。同一フラグの重複・両フラグの併用・未知フラグはいずれも、後勝ちで黙殺せずエラーにします。
func parseFilter(args []string) (*bool, error) {
	seenActive, seenInactive := false, false
	for _, a := range args {
		switch a {
		case activeOnlyFlag:
			if seenActive {
				return nil, xerrors.Wrap(errDuplicateFlag, activeOnlyFlag)
			}
			seenActive = true
		case inactiveOnlyFlag:
			if seenInactive {
				return nil, xerrors.Wrap(errDuplicateFlag, inactiveOnlyFlag)
			}
			seenInactive = true
		default:
			return nil, xerrors.Wrap(errUnknownFlag, a)
		}
	}
	if seenActive && seenInactive {
		return nil, xerrors.Wrap(errConflictingFilterFlags, activeOnlyFlag+", "+inactiveOnlyFlag)
	}
	switch {
	case seenActive:
		return new(true), nil
	case seenInactive:
		return new(false), nil
	default:
		return nil, nil //nolint:nilnil // フラグ未指定は「絞り込みなし（全件）」を表すため nil, nil が正常値
	}
}

// filterLabel は、絞り込み条件をログ用のラベルへ変換します（nil→all / true→active / false→inactive）。
func filterLabel(active *bool) string {
	switch {
	case active == nil:
		return "all"
	case *active:
		return "active"
	default:
		return "inactive"
	}
}
