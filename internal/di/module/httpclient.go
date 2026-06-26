package module

import (
	"go-boilerplate/internal/infrastructure/httpclient"

	"go.uber.org/fx"
)

// HTTPClientProfilesIn は、httpclient_profiles グループに集まった各 Downstream の Profile を
// 集約する入力です（グループの consumer 側。di/job の RunnerIn と同形）。
type HTTPClientProfilesIn struct {
	fx.In

	Profiles []httpclient.DownstreamProfile `group:"httpclient_profiles"`
}

// provideHTTPClientRegistry は、グループに集まった Profile から Registry を生成します
// （di/job の ProvideRunner と同形）。
func provideHTTPClientRegistry(in HTTPClientProfilesIn) httpclient.Registry {
	return httpclient.NewRegistryFromProfiles(in.Profiles)
}

// httpClientModule は、resilient な外部 HTTP client substrate を提供するfx.Moduleです。
// httpclient_profiles グループに集まった各 Downstream の Profile を registry がまとめて解決します。
func httpClientModule() fx.Option {
	return fx.Module("httpclient",
		fx.Provide(
			provideHTTPClientRegistry,
			httpclient.New,
		),
	)
}

// provideHTTPClientProfiles は、各 Downstream の Profile コンストラクタを httpclient_profiles
// グループへ登録します（producer 側）。provideJobs / provideWorkers と同形の
// 「グループへ寄与するコンストラクタ群」をまとめるヘルパーで、各所での fx.Annotate 重複を排します。
func provideHTTPClientProfiles(constructors ...any) fx.Option {
	opts := make([]fx.Option, len(constructors))
	for i, c := range constructors {
		opts[i] = fx.Provide(
			fx.Annotate(
				c,
				fx.ResultTags(`group:"httpclient_profiles"`),
			),
		)
	}
	return fx.Options(opts...)
}
