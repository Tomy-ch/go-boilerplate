package module

import (
	"go-boilerplate/internal/controller/job"
	"go-boilerplate/internal/controller/job/idempotencygc"
	"go-boilerplate/internal/controller/job/outboxgc"
	"go-boilerplate/internal/controller/job/productimagegc" // sample-api:line
	"go-boilerplate/internal/controller/job/usercount"      // sample-api:line
	"go-boilerplate/internal/controller/job/userpurge"      // sample-api:line
	dijob "go-boilerplate/internal/di/job"
	"go-boilerplate/internal/di/job/hook"

	"go.uber.org/fx"
)

// JobModule は、バックグラウンドジョブ関連の依存関係を提供するfx.Moduleです。
func JobModule() fx.Option {
	return fx.Module("job",
		provideJobs(
			idempotencygc.New,
			outboxgc.New,
			productimagegc.New, // sample-api:line
			usercount.New,      // sample-api:line
			userpurge.New,      // sample-api:line
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
