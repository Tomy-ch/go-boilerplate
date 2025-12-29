package module

import (
	"boilerplate-go/internal/controller/job/usercount"
	"boilerplate-go/internal/di/job"

	"go.uber.org/fx"
)

func JobModule() fx.Option {
	return fx.Module("job",
		provideJobs(
			// ここにジョブのコンストラクタを追加します。
			usercount.New,
		),
		fx.Provide(
			job.ProvideRunner,
			job.NewState,
		),
		fx.Invoke(job.RegisterJobHooks),
	)
}

// provideJobs は、ジョブのコンストラクタをfxのオプションとして提供します。
func provideJobs(constructors ...any) fx.Option {
	opts := make([]fx.Option, 0, len(constructors))
	for _, c := range constructors {
		opts = append(opts, fx.Provide(
			fx.Annotate(
				c,
				fx.ResultTags(`group:"jobs"`),
			),
		))
	}
	return fx.Options(opts...)
}
