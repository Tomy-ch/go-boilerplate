package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	infrasystem "go-boilerplate/internal/system"
)

// 各ケースは fx アプリを起動し、ObservabilityModule 経由で buildinfo を prometheus の
// DefaultRegisterer（global）へ登録する。複数アプリを並列起動すると同一 global への
// 同時アクセスとなり -race が競合を検出するため、本テストは非並列にする（本番は単一アプリ）。
//
//nolint:paralleltest // 上記の理由により fx アプリ起動ケースは非並列
func Test_httpClientModule_ProvidesClient(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("実モジュール配線(clock + httpclient)から Client が構築される", func(t *testing.T) {
			var client httpclient.Client
			app := newHTTPClientTestApp(t, fx.Populate(&client))

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, client)
		})

		t.Run("required と対応 profile が揃っていれば起動する", func(t *testing.T) {
			var client httpclient.Client
			app := newHTTPClientTestApp(t,
				provideHTTPClientProfiles(func() httpclient.DownstreamProfile {
					return httpclient.DownstreamProfile{Name: "svc", Profile: httpclient.DefaultProfile()}
				}),
				provideRequiredDownstreams(func() httpclient.Downstream { return "svc" }),
				fx.Populate(&client),
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, client)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("required に対応する profile が未登録の場合、起動に失敗する", func(t *testing.T) {
			// profile を登録せず required だけ宣言すると、silent fallback ではなく起動失敗になる。
			var client httpclient.Client
			app := newHTTPClientTestApp(t,
				provideRequiredDownstreams(func() httpclient.Downstream { return "orphan" }),
				fx.Populate(&client),
			)

			require.ErrorIs(t, app.Start(context.Background()), errRequiredProfileMissing)
		})
	})
}

func Test_provideHTTPClientRegistry(t *testing.T) {
	t.Parallel()

	profile := func(name httpclient.Downstream) httpclient.DownstreamProfile {
		return httpclient.DownstreamProfile{Name: name, Profile: httpclient.DefaultProfile()}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("required が全て登録済みの場合は Profile を引ける Registry を返す", func(t *testing.T) {
			t.Parallel()

			in := HTTPClientProfilesIn{
				Profiles: []httpclient.DownstreamProfile{profile("svc")},
				Required: []httpclient.Downstream{"svc"},
			}

			reg, err := provideHTTPClientRegistry(in)

			require.NoError(t, err)
			require.NotNil(t, reg)
			assert.Equal(t, httpclient.DefaultProfile(), reg.Profile("svc"))
		})

		t.Run("required が空の場合も Registry を返す", func(t *testing.T) {
			t.Parallel()

			reg, err := provideHTTPClientRegistry(HTTPClientProfilesIn{})

			require.NoError(t, err)
			assert.NotNil(t, reg)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("required に対応する profile が無い場合は欠落名を含むエラーを返す", func(t *testing.T) {
			t.Parallel()

			in := HTTPClientProfilesIn{
				Profiles: []httpclient.DownstreamProfile{profile("svc")},
				Required: []httpclient.Downstream{"svc", "orphan"},
			}

			reg, err := provideHTTPClientRegistry(in)

			require.ErrorContains(t, err, "orphan")
			assert.Nil(t, reg)
		})

		t.Run("同一 Downstream の profile が重複登録された場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			in := HTTPClientProfilesIn{
				Profiles: []httpclient.DownstreamProfile{profile("svc"), profile("svc")},
			}

			reg, err := provideHTTPClientRegistry(in)

			require.ErrorContains(t, err, "duplicate")
			assert.Nil(t, reg)
		})
	})
}

func Test_provideHTTPClientProfiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した全コンストラクタの Profile が httpclient_profiles グループへ集まる", func(t *testing.T) {
			t.Parallel()

			got := collectGroup[httpclient.DownstreamProfile](t, `group:"httpclient_profiles"`, provideHTTPClientProfiles(
				func() httpclient.DownstreamProfile {
					return httpclient.DownstreamProfile{Name: "a", Profile: httpclient.DefaultProfile()}
				},
				func() httpclient.DownstreamProfile {
					return httpclient.DownstreamProfile{Name: "b", Profile: httpclient.DefaultProfile()}
				},
			))

			names := make([]httpclient.Downstream, 0, len(got))
			for _, p := range got {
				names = append(names, p.Name)
			}
			assert.ElementsMatch(t, []httpclient.Downstream{"a", "b"}, names)
		})

		t.Run("コンストラクタが 0 個の場合は何も登録しない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, collectGroup[httpclient.DownstreamProfile](t, `group:"httpclient_profiles"`, provideHTTPClientProfiles()))
		})
	})
}

func Test_provideRequiredDownstreams(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した全コンストラクタの Downstream が required_downstreams グループへ集まる", func(t *testing.T) {
			t.Parallel()

			got := collectGroup[httpclient.Downstream](t, `group:"required_downstreams"`, provideRequiredDownstreams(
				func() httpclient.Downstream { return "a" },
				func() httpclient.Downstream { return "b" },
			))

			assert.ElementsMatch(t, []httpclient.Downstream{"a", "b"}, got)
		})

		t.Run("コンストラクタが 0 個の場合は何も登録しない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, collectGroup[httpclient.Downstream](t, `group:"required_downstreams"`, provideRequiredDownstreams()))
		})
	})
}

// newHTTPClientTestApp は、本番同様の clockModule / httpClientModule 配線に extra を足した fx アプリを構築します。
func newHTTPClientTestApp(t *testing.T, extra ...fx.Option) *fx.App {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
	mockLog := mock_logging.NewMockLogger(ctrl)
	mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)
	mockReg.EXPECT().RegisterStop(gomock.Any()).AnyTimes()

	return fx.New(append([]fx.Option{
		ObservabilityModule(),
		clockModule(),
		httpClientModule(),
		fx.Provide(func() testing.TB { return t }),
		fx.Provide(func() lifecycle.Registrar { return mockReg }),
		fx.Provide(func() logging.Logger { return mockLog }),
		fx.Provide(func() logging.LogFieldBuilder { return mockLF }),
		fx.Provide(func() *config.ApplicationConfig {
			return config.NewApplicationConfig(config.MockConfigForTest(t))
		}),
		fx.Provide(func() *config.ObservabilityConfig {
			return config.NewObservabilityConfig(config.MockConfigForTest(t))
		}),
		fx.Provide(infrasystem.NewBuildInfo),
		fx.NopLogger,
	}, extra...)...)
}

// value group 経由で登録した profile が registry へ反映されるかを見るため実アプリを起動する。
// 起動を伴うケースは global registerer 競合を避けるため非並列にする。
//
//nolint:paralleltest // 上記の理由により fx アプリ起動ケースは非並列
func Test_httpClientModule(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("value group に集めた profile を引ける Registry を提供する", func(t *testing.T) {
			var registry httpclient.Registry
			app := newHTTPClientTestApp(t,
				provideHTTPClientProfiles(func() httpclient.DownstreamProfile {
					return httpclient.DownstreamProfile{Name: "svc", Profile: httpclient.DefaultProfile()}
				}),
				fx.Populate(&registry),
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })

			require.NotNil(t, registry)
			assert.Equal(t, httpclient.DefaultProfile(), registry.Profile("svc"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("未配線では Registry が解決できずグラフ検証に失敗する", func(t *testing.T) {
			var registry httpclient.Registry

			opts := append(commonDeps(), clockModule(), fx.Populate(&registry), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
