package module

import (
	"boilerplate-go/internal/controller/job"
	"boilerplate-go/internal/controller/job/usercount"
	dijob "boilerplate-go/internal/di/job"
	"boilerplate-go/internal/di/job/hook"

	"go.uber.org/fx"
)

func JobModule() fx.Option {
	return fx.Module("job",
		provideJobs(
			// ここにジョブのコンストラクタを追加します。
			usercount.New,
		),
		fx.Provide(
			dijob.ProvideRunner,
			job.NewState,
		),
		fx.Invoke(hook.RegisterJobHooks),
	)
}

// provideJobs は、ジョブのコンストラクタをfxのオプションとして提供します。
func provideJobs(constructors ...any) fx.Option {
	opts := make([]fx.Option, len(constructors))
	for i, c := range constructors {
		opts[i] = fx.Provide(
			fx.Annotate(
				c,
				fx.ResultTags(`group:"jobs"`),
			),
		)
	}
	return fx.Options(opts...)
}
