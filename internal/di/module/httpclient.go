package module

import (
	"fmt"

	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/pkg/xerrors"

	"go.uber.org/fx"
)

// HTTPClientProfilesIn は、httpclient_profiles グループに集まった各 Downstream の Profile と、
// required_downstreams グループに集まった「登録が必須な Downstream」を集約する入力です。
type HTTPClientProfilesIn struct {
	fx.In

	Profiles []httpclient.DownstreamProfile `group:"httpclient_profiles"`
	Required []httpclient.Downstream        `group:"required_downstreams"`
}

// provideHTTPClientRegistry は、グループに集まった Profile から Registry を生成します。
// 各 gateway が宣言した required Downstream に対応する Profile が欠けている場合は起動時に失敗させ、
// 登録漏れが silent な DefaultProfile fallback へ流れるのを防ぎます。
func provideHTTPClientRegistry(in HTTPClientProfilesIn) (httpclient.Registry, error) {
	if missing := httpclient.MissingDownstreams(in.Profiles, in.Required); len(missing) > 0 {
		return nil, xerrors.New(fmt.Sprintf("httpclient profile missing for required downstreams: %v", missing))
	}
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
// グループへ登録します（producer 側）。各コンストラクタへの fx.Annotate 重複を排するヘルパーです。
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

// provideRequiredDownstreams は、各 constructors が返す Downstream を required_downstreams グループへ
// 登録します。ここへ登録された Downstream は、対応 profile が未登録のまま起動すると loud な失敗になります。
func provideRequiredDownstreams(constructors ...any) fx.Option {
	opts := make([]fx.Option, len(constructors))
	for i, c := range constructors {
		opts[i] = fx.Provide(
			fx.Annotate(
				c,
				fx.ResultTags(`group:"required_downstreams"`),
			),
		)
	}
	return fx.Options(opts...)
}
