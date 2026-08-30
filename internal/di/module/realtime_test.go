package module

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/ctxhelper"
	oapiauth "go-boilerplate/internal/controller/httpstack/oapi/auth"
	ctrlrealtime "go-boilerplate/internal/controller/realtime"
	mock_ctrlrealtime "go-boilerplate/internal/controller/realtime/mock"
	"go-boilerplate/internal/controller/stream"
	streamauth "go-boilerplate/internal/controller/stream/auth"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/server/hook"
	"go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	realtimeinfra "go-boilerplate/internal/infrastructure/realtime"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_rt "go-boilerplate/internal/usecase/boundary/realtime/mock"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	mock_realtime "go-boilerplate/internal/usecase/realtime/mock"
	"go-boilerplate/pkg/xerrors"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/mock/gomock"
)

// realtimeDeps は、realtime module の graph 検証に要る下位モジュール群です（clock は infrastructure 側の module）。
// stream handler の登録に要る *echo.Echo だけは server module の代わりにここで供給します。
func realtimeDeps() []fx.Option {
	return append(commonDeps(), clockModule(), realtimeModule(), fx.Provide(echo.New))
}

// realtimeRunDeps は、constructor を実際に走らせて value group を集めるための依存です。ConfigModule /
// ObservabilityModule / DatabaseModule は global（設定の読み込みと prometheus の既定 registry）へ触るため、
// 並列テストと競合する。realtime が要るのは設定 3 つと Logger / TracerFactory / outbound クライアントだけ
// なので、いずれも test 用の実装を直接差し込む。
func realtimeRunDeps(t *testing.T) fx.Option {
	t.Helper()

	cfg := config.MockConfigForTest(t)

	return fx.Options(
		lifecycle.Module(), clockModule(), realtimeModule(),
		fx.Provide(func() *config.ApplicationConfig { return config.NewApplicationConfig(cfg) }),
		fx.Provide(func() *config.EndpointConfig { return config.NewEndpointConfig(cfg) }),
		fx.Provide(func() *config.RealtimeConfig { return config.NewRealtimeConfig(cfg) }),
		fx.Provide(func() logging.Logger { return logging.NewTestLogger(t) }),
		fx.Provide(func() observability.TracerFactory { return observability.NewNoopTracerFactory(t) }),
		fx.Provide(func() *observability.OutboundHTTPClient { return observability.NewDisabledOutboundHTTPClient(true) }),
		fx.Provide(echo.New),
		fx.Replace(testFanout(t)),
	)
}

// testFanout は、emulator の endpoint を向いた fan-out の依存を返します（接続はしない）。
func testFanout(t *testing.T) realtimeFanout {
	t.Helper()

	clients, err := realtimeinfra.NewClients(t.Context(), realtimeinfra.ClientConfig{
		Endpoint: "http://localhost:4100", Region: "us-east-1", AccessKeyID: "k", SecretAccessKey: "s",
	})
	require.NoError(t, err)

	return realtimeFanout{clients: clients, topicARN: "arn:aws:sns:us-east-1:000000000000:realtime-test"}
}

func Test_realtimeModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	validateGraph(t, realtimeDeps()...)
}

func Test_realtimeModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("3 つの store と機構側 usecase と StreamTicket scheme の認証器を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				log      rt.EventLogStore
				tickets  rt.StreamTicketStore
				leases   rt.InstanceLeaseStore
				secrets  rt.SecretGenerator
				cursor   ucrealtime.CursorValidator
				issuer   ucrealtime.TicketIssuer
				verifier ucrealtime.TicketVerifier
			)

			validateGraph(
				t,
				append(
					realtimeDeps(),
					fx.Populate(&log, &tickets, &leases, &secrets, &cursor, &issuer, &verifier),
				)...)

			// value group は空でも解決するため、group への登録は実体を集めて数える。
			schemes := collectGroup[oapiauth.SchemeAuthenticator](t, `group:"oapi.security.schemes"`, realtimeRunDeps(t))
			assert.Len(t, schemes, 1)
		})

		t.Run("fan-out の publish 側・受信側・lifecycle の参加者を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				notifier  rt.RevocationNotifier
				revoker   ucrealtime.AccessRevoker
				id        rt.InstanceID
				attrs     realtimeinfra.AttributesBuilder
				sub       rt.InstanceSubscription
				keeper    ucrealtime.LeaseKeeper
				engine    *ctrlrealtime.Engine
				heartbeat *ctrlrealtime.Heartbeat
			)

			validateGraph(t, append(realtimeDeps(),
				fx.Populate(&notifier, &revoker, &id, &attrs, &sub, &keeper, &engine, &heartbeat))...)

			// value group は空でも解決するため、参加者は実体を集めて名前まで見る。
			deps := realtimeRunDeps(t)
			probes := collectGroup[hook.StartupProbe](t, `group:"serve.startup"`, deps)
			provisioners := collectGroup[hook.Provisioner](t, `group:"serve.provisioners"`, deps)
			runners := collectGroup[hook.Runner](t, `group:"serve.runners"`, deps)
			drainers := collectGroup[hook.Drainer](t, `group:"serve.drainers"`, deps)

			require.Len(t, probes, 1)
			require.Len(t, provisioners, 1)
			require.Len(t, runners, 2)
			require.Len(t, drainers, 1)
			assert.Equal(t, realtimeParticipantName, probes[0].Name)
			assert.Equal(t, realtimeParticipantName, provisioners[0].Name)
			assert.ElementsMatch(t, []string{"realtime-consumer", "realtime-heartbeat"}, []string{runners[0].Name, runners[1].Name})
			assert.Equal(t, realtimeParticipantName+"-stream", drainers[0].Name)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では EventLogStore が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var log rt.EventLogStore

			opts := append(commonDeps(), fx.Populate(&log), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}

func Test_provideRealtimeClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定の endpoint と資格情報からクライアントを組み立てる", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			c, err := provideRealtimeClient(
				config.NewRealtimeConfig(
					mock,
				),
				config.NewEndpointConfig(mock),
				observability.NewDisabledOutboundHTTPClient(true),
			)
			require.NoError(t, err)
			assert.Nil(t, c.Options().BaseEndpoint, "テスト設定の endpoint は空＝SDK 既定の解決")
			assert.Equal(t, config.NewRealtimeConfig(mock).Region(), c.Options().Region)
		})
	})
}

func Test_provideEventLogStore(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	assert.NotNil(t, provideEventLogStore(testkit.NewTestClient(t), cfg, observability.NewNoopTracerFactory(t)))
}

func Test_provideStreamTicketStore(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	assert.NotNil(t, provideStreamTicketStore(testkit.NewTestClient(t), cfg, observability.NewNoopTracerFactory(t)))
}

func Test_provideInstanceLeaseStore(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	assert.NotNil(t, provideInstanceLeaseStore(testkit.NewTestClient(t), cfg, observability.NewNoopTracerFactory(t)))
}

func Test_provideRealtimeSecretGenerator(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, provideRealtimeSecretGenerator())
}

func Test_provideCursorValidator(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	log := provideEventLogStore(testkit.NewTestClient(t), cfg, observability.NewNoopTracerFactory(t))
	assert.NotNil(
		t,
		provideCursorValidator(
			log,
			mock_clock.NewMockClock(gomock.NewController(t)),
			observability.NewNoopTracerFactory(t),
		),
	)
}

func Test_provideTicketIssuer(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	store := provideStreamTicketStore(testkit.NewTestClient(t), cfg, observability.NewNoopTracerFactory(t))
	clk := mock_clock.NewMockClock(gomock.NewController(t))
	assert.NotNil(
		t,
		provideTicketIssuer(store, provideRealtimeSecretGenerator(), clk, observability.NewNoopTracerFactory(t)),
	)
}

func Test_provideTicketVerifier(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	store := provideStreamTicketStore(testkit.NewTestClient(t), cfg, observability.NewNoopTracerFactory(t))
	assert.NotNil(
		t,
		provideTicketVerifier(
			store,
			mock_clock.NewMockClock(gomock.NewController(t)),
			observability.NewNoopTracerFactory(t),
		),
	)
}

func Test_provideStreamTicketScheme(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("StreamTicket scheme を担当する認証器を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			v := mock_realtime.NewMockTicketVerifier(ctrl)
			v.EXPECT().Verify(gomock.Any(), "raw", rt.StreamID("d")).Return(rt.StreamGrant{Subject: "s"}, nil)

			s := provideStreamTicketScheme(v)
			assert.Equal(t, streamauth.SchemeName, s.Scheme())

			// 受け取った verifier がそのまま認証器に渡っていること（検証が mock に届く）。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/streams/d?ticket=raw", nil)
			req = req.WithContext(ctxhelper.WithStreamGrant(req.Context()))
			in := &openapi3filter.AuthenticationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{
					Request:    req,
					PathParams: map[string]string{"destination": "d"},
				},
				SecuritySchemeName: streamauth.SchemeName,
				SecurityScheme:     &openapi3.SecurityScheme{Type: "apiKey", In: "query", Name: "ticket"},
			}
			require.NoError(t, s.Authenticate(context.Background(), in))
		})
	})
}

func Test_provideRealtimeFanout(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("topic の ARN と SNS / SQS の endpoint を設定から写し、SDK は outbound クライアントで呼ぶ", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			cfg := config.NewRealtimeConfig(mock)
			epCfg := config.NewEndpointConfig(mock)
			cfg.SetTopic(t, "arn:aws:sns:us-east-1:000000000000:realtime")
			epCfg.SetRealtimePubSub(t, "http://localhost:4100")
			outbound := observability.NewDisabledOutboundHTTPClient(true)

			f, err := provideRealtimeFanout(cfg, epCfg, outbound)
			require.NoError(t, err)
			assert.Equal(t, "arn:aws:sns:us-east-1:000000000000:realtime", f.topicARN)
			assert.Equal(t, "http://localhost:4100", awssdk.ToString(f.clients.SNS.Options().BaseEndpoint))
			assert.Equal(t, "http://localhost:4100", awssdk.ToString(f.clients.SQS.Options().BaseEndpoint))
			assert.Equal(t, cfg.Region(), f.clients.SNS.Options().Region)
			assert.Same(t, outbound, f.clients.SNS.Options().HTTPClient)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テスト設定は topic の ARN が空なので ErrRealtimeTopicNotConfigured（fail-closed）", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			_, err := provideRealtimeFanout(
				config.NewRealtimeConfig(mock),
				config.NewEndpointConfig(mock),
				observability.NewDisabledOutboundHTTPClient(true),
			)
			require.ErrorIs(t, err, ErrRealtimeTopicNotConfigured)
		})
	})
}

func Test_newRealtimeFanout(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("topic の ARN と接続設定からクライアントを組み立てる", func(t *testing.T) {
			t.Parallel()

			f, err := newRealtimeFanout(t.Context(), "arn:aws:sns:us-east-1:000000000000:realtime-test", realtimeinfra.ClientConfig{
				Endpoint: "http://localhost:4100", Region: "us-east-1", AccessKeyID: "k", SecretAccessKey: "s",
			})
			require.NoError(t, err)
			assert.Equal(t, "arn:aws:sns:us-east-1:000000000000:realtime-test", f.topicARN)
			assert.NotNil(t, f.clients.SNS)
			assert.NotNil(t, f.clients.SQS)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("topic の ARN が空なら ErrRealtimeTopicNotConfigured", func(t *testing.T) {
			t.Parallel()

			_, err := newRealtimeFanout(t.Context(), "", realtimeinfra.ClientConfig{Region: "us-east-1"})
			require.ErrorIs(t, err, ErrRealtimeTopicNotConfigured)
		})

		t.Run("資格情報が片方だけなら構築に失敗する", func(t *testing.T) {
			t.Parallel()

			_, err := newRealtimeFanout(t.Context(), "arn:topic", realtimeinfra.ClientConfig{Region: "us-east-1", AccessKeyID: "k"})
			require.Error(t, err)
		})
	})
}

func Test_provideRevocationNotifier(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, provideRevocationNotifier(testFanout(t), observability.NewNoopTracerFactory(t)))
}

func Test_provideAccessRevoker(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	assert.NotNil(
		t,
		provideAccessRevoker(
			mock_rt.NewMockStreamTicketStore(ctrl),
			mock_rt.NewMockRevocationNotifier(ctrl),
			observability.NewNoopTracerFactory(t),
		),
	)
}

func Test_provideInstanceID(t *testing.T) {
	t.Parallel()

	a, err := provideInstanceID()
	require.NoError(t, err)
	b, err := provideInstanceID()
	require.NoError(t, err)

	assert.Regexp(t, `^[0-9a-f-]{36}$`, string(a))
	assert.NotEqual(t, a, b, "起動ごとに別の識別子")
}

func Test_provideInstanceQueueAttributes(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// buildFor は、env の実装が組み立てる属性を返す。
		buildFor := func(t *testing.T, env string) map[string]string {
			t.Helper()

			attrs, err := provideInstanceQueueAttributes(newAppCfgForEnv(t, env), cfg)
			require.NoError(t, err)

			built, err := attrs.Build("arn:q")
			require.NoError(t, err)

			return built
		}

		t.Run("local は timings だけの属性", func(t *testing.T) {
			t.Parallel()

			built := buildFor(t, config.EnvLocal)
			assert.NotContains(t, built, "Policy")
			assert.Contains(t, built, "VisibilityTimeout")
		})

		t.Run("ci は timings だけの属性", func(t *testing.T) {
			t.Parallel()

			built := buildFor(t, config.EnvCI)
			assert.NotContains(t, built, "Policy")
			assert.Contains(t, built, "VisibilityTimeout")
		})

		t.Run("test は timings だけの属性", func(t *testing.T) {
			t.Parallel()

			built := buildFor(t, config.EnvTest)
			assert.NotContains(t, built, "Policy")
			assert.Contains(t, built, "VisibilityTimeout")
		})

		t.Run("dast は timings だけの属性", func(t *testing.T) {
			t.Parallel()

			built := buildFor(t, config.EnvDast)
			assert.NotContains(t, built, "Policy")
			assert.Contains(t, built, "VisibilityTimeout")
		})

		t.Run("dev は policy を含む全属性", func(t *testing.T) {
			t.Parallel()

			built := buildFor(t, config.EnvDevelopment)
			assert.Contains(t, built, "Policy")
			assert.Contains(t, built, "SqsManagedSseEnabled")
		})

		t.Run("stg は policy を含む全属性", func(t *testing.T) {
			t.Parallel()

			built := buildFor(t, config.EnvStaging)
			assert.Contains(t, built, "Policy")
			assert.Contains(t, built, "SqsManagedSseEnabled")
		})

		t.Run("prd は policy を含む全属性", func(t *testing.T) {
			t.Parallel()

			built := buildFor(t, config.EnvProduction)
			assert.Contains(t, built, "Policy")
			assert.Contains(t, built, "SqsManagedSseEnabled")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("名指ししない環境は ErrInvalidArgument", func(t *testing.T) {
			t.Parallel()

			_, err := provideInstanceQueueAttributes(newAppCfgForEnv(t, "unknown"), cfg)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_provideInstanceSubscription(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	assert.NotNil(
		t,
		provideInstanceSubscription(
			testFanout(t),
			cfg,
			realtimeinfra.NewEmulatorQueueAttributes(),
			observability.NewNoopTracerFactory(t),
		),
	)
}

func Test_provideLeaseKeeper(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	assert.NotNil(
		t,
		provideLeaseKeeper(
			mock_rt.NewMockInstanceLeaseStore(ctrl),
			mock_clock.NewMockClock(ctrl),
			observability.NewNoopTracerFactory(t),
		),
	)
}

func Test_provideRealtimeConsumer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	engine := provideRealtimeConsumer(
		mock_rt.NewMockInstanceSubscription(ctrl),
		mock_ctrlrealtime.NewMockReprovisioner(ctrl),
		mock_ctrlrealtime.NewMockWaker(ctrl), mock_ctrlrealtime.NewMockRevoker(ctrl),
		mock_clock.NewMockSleeper(ctrl),
		logging.NewTestLogger(t), observability.NewNoopTracerFactory(t),
	)
	assert.NotNil(t, engine)
}

func Test_provideRealtimeHeartbeat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	assert.NotNil(t, provideRealtimeHeartbeat(
		mock_realtime.NewMockLeaseKeeper(
			ctrl,
		),
		"inst-1",
		mock_clock.NewMockSleeper(ctrl),
		logging.NewTestLogger(t),
		observability.NewNoopTracerFactory(t),
	))
}

func Test_provideRealtimeStartupProbe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("EventLog の読み取りが通れば到達可能", func(t *testing.T) {
			t.Parallel()

			log := mock_rt.NewMockEventLogStore(gomock.NewController(t))
			log.EXPECT().Latest(gomock.Any(), startupProbeStreamID).Return(rt.DeliveryEvent{}, false, nil)

			p := provideRealtimeStartupProbe(log)
			assert.Equal(t, realtimeParticipantName, p.Name)
			require.NoError(t, p.Probe(t.Context()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("store に届かなければその失敗を返す", func(t *testing.T) {
			t.Parallel()

			log := mock_rt.NewMockEventLogStore(gomock.NewController(t))
			log.EXPECT().
				Latest(gomock.Any(), startupProbeStreamID).
				Return(rt.DeliveryEvent{}, false, apperror.ErrUnavailable)

			require.ErrorIs(t, provideRealtimeStartupProbe(log).Probe(t.Context()), apperror.ErrUnavailable)
		})
	})
}

func Test_provideRealtimeProvisioner(t *testing.T) {
	t.Parallel()

	errBoom := xerrors.New("boom")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動は lease → 受信先、片付けは 受信先 → lease の順", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sub := mock_rt.NewMockInstanceSubscription(ctrl)
			keeper := mock_realtime.NewMockLeaseKeeper(ctrl)
			gomock.InOrder(
				keeper.EXPECT().Beat(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
				sub.EXPECT().Provision(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
				sub.EXPECT().Teardown(gomock.Any()).Return(nil),
				keeper.EXPECT().Release(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
			)

			p := provideRealtimeProvisioner(sub, keeper, "inst-1")
			assert.Equal(t, realtimeParticipantName, p.Name)
			require.NoError(t, p.Provision(t.Context()))
			require.NoError(t, p.Teardown(t.Context()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lease が書けなければ受信先を作らない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sub := mock_rt.NewMockInstanceSubscription(ctrl)
			keeper := mock_realtime.NewMockLeaseKeeper(ctrl)
			keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).Return(errBoom)

			require.ErrorIs(t, provideRealtimeProvisioner(sub, keeper, "inst-1").Provision(t.Context()), errBoom)
		})

		t.Run("受信先を作れなければ、片付けてから lease を取り消して失敗する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sub := mock_rt.NewMockInstanceSubscription(ctrl)
			keeper := mock_realtime.NewMockLeaseKeeper(ctrl)
			gomock.InOrder(
				keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).Return(nil),
				sub.EXPECT().Provision(gomock.Any(), gomock.Any()).Return(errBoom),
				sub.EXPECT().Teardown(gomock.Any()).Return(nil),
				keeper.EXPECT().Release(gomock.Any(), gomock.Any()).Return(nil),
			)

			require.ErrorIs(t, provideRealtimeProvisioner(sub, keeper, "inst-1").Provision(t.Context()), errBoom)
		})

		t.Run("受信先を作れず片付けにも失敗したら lease は残す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sub := mock_rt.NewMockInstanceSubscription(ctrl)
			keeper := mock_realtime.NewMockLeaseKeeper(ctrl)
			gomock.InOrder(
				keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).Return(nil),
				sub.EXPECT().Provision(gomock.Any(), gomock.Any()).Return(errBoom),
				sub.EXPECT().Teardown(gomock.Any()).Return(apperror.ErrUnavailable),
			)

			err := provideRealtimeProvisioner(sub, keeper, "inst-1").Provision(t.Context())
			require.ErrorIs(t, err, errBoom)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("片付けに失敗したら lease は残す（orphan cleanup が辿れるようにする）", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sub := mock_rt.NewMockInstanceSubscription(ctrl)
			keeper := mock_realtime.NewMockLeaseKeeper(ctrl)
			sub.EXPECT().Teardown(gomock.Any()).Return(errBoom)

			require.ErrorIs(t, provideRealtimeProvisioner(sub, keeper, "inst-1").Teardown(t.Context()), errBoom)
		})

		t.Run("片付けの後に lease の取り消しが失敗すればその失敗を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sub := mock_rt.NewMockInstanceSubscription(ctrl)
			keeper := mock_realtime.NewMockLeaseKeeper(ctrl)
			sub.EXPECT().Teardown(gomock.Any()).Return(nil)
			keeper.EXPECT().Release(gomock.Any(), gomock.Any()).Return(apperror.ErrUnavailable)

			require.ErrorIs(t, provideRealtimeProvisioner(sub, keeper, "inst-1").Teardown(t.Context()), apperror.ErrUnavailable)
		})
	})
}

func Test_provideRealtimeConsumerRunner(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	sub := mock_rt.NewMockInstanceSubscription(ctrl)
	ran := make(chan struct{})
	var once sync.Once
	sub.EXPECT().
		Receive(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ int) ([]rt.Notification, error) {
			once.Do(func() { close(ran) })
			<-ctx.Done()

			return nil, ctx.Err()
		}).
		MinTimes(1)
	engine := provideRealtimeConsumer(
		sub,
		mock_ctrlrealtime.NewMockReprovisioner(ctrl),
		mock_ctrlrealtime.NewMockWaker(ctrl),
		mock_ctrlrealtime.NewMockRevoker(ctrl),
		mock_clock.NewMockSleeper(ctrl),
		logging.NewTestLogger(t),
		observability.NewNoopTracerFactory(t),
	)

	r := provideRealtimeConsumerRunner(engine)
	assert.Equal(t, "realtime-consumer", r.Name)

	start, stop := r.Runner.Bind()
	require.NoError(t, start(t.Context()))
	<-ran // Body が engine を回している
	require.NoError(t, stop(t.Context()))
}

func Test_provideRealtimeHeartbeatRunner(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	keeper := mock_realtime.NewMockLeaseKeeper(ctrl)
	ran := make(chan struct{})
	var once sync.Once
	keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, rt.InstanceID) error {
		once.Do(func() { close(ran) })

		return nil
	}).MinTimes(1)
	sleeper := mock_clock.NewMockSleeper(ctrl)
	sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, _ time.Duration) error {
		<-ctx.Done()

		return ctx.Err()
	}).AnyTimes()
	heartbeat := provideRealtimeHeartbeat(
		keeper,
		"inst-1",
		sleeper,
		logging.NewTestLogger(t),
		observability.NewNoopTracerFactory(t),
	)

	r := provideRealtimeHeartbeatRunner(heartbeat)
	assert.Equal(t, "realtime-heartbeat", r.Name)

	start, stop := r.Runner.Bind()
	require.NoError(t, start(t.Context()))
	<-ran // Body が heartbeat を回している
	require.NoError(t, stop(t.Context()))
}

// newTestRegistry は、provider が返す registry を組み立てます（依存はすべて test 用）。
func newTestRegistry(t *testing.T) *stream.Registry {
	t.Helper()

	ctrl := gomock.NewController(t)

	return provideConnectionRegistry(
		mock_realtime.NewMockReplayer(ctrl),
		mock_clock.NewMockSleeper(ctrl),
		logging.NewTestLogger(t),
		observability.NewNoopTracerFactory(t),
	)
}

func Test_provideReplayer(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, provideReplayer(
		mock_rt.NewMockEventLogStore(gomock.NewController(t)),
		observability.NewNoopTracerFactory(t),
	))
}

func Test_provideConnectionRegistry(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, newTestRegistry(t))
}

func Test_provideStreamer(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	assert.Same(t, registry, provideStreamer(registry), "registry がそのまま Streamer として出ること")
}

func Test_provideWaker(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	assert.Same(t, registry, provideWaker(registry), "Streamer と同じ registry が wakeup の受け口になること")
}

func Test_provideRevoker(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	assert.Same(t, registry, provideRevoker(registry), "Streamer と同じ registry が失効の受け口になること")
}

func Test_provideRealtimeStreamDrainer(t *testing.T) {
	t.Parallel()

	d := provideRealtimeStreamDrainer(newTestRegistry(t))

	assert.Equal(t, "realtime-stream", d.Name)
	require.NotNil(t, d.Drain)
	require.NoError(t, d.Drain(t.Context()), "接続が 1 本も無ければ drain は即座に終わる")
}

func Test_provideRealtimeReprovisioner(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lease を書き直してから受信先を作り直す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sub := mock_rt.NewMockInstanceSubscription(ctrl)
			keeper := mock_realtime.NewMockLeaseKeeper(ctrl)

			// この順序が逆になると、lease に指されない queue ができて orphan cleanup から辿れなくなる。
			gomock.InOrder(
				keeper.EXPECT().Beat(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
				sub.EXPECT().Provision(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
			)

			require.NoError(t, provideRealtimeReprovisioner(sub, keeper, "inst-1").Reprovision(t.Context()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lease を書き直せなければ受信先を作らない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sub := mock_rt.NewMockInstanceSubscription(ctrl)
			keeper := mock_realtime.NewMockLeaseKeeper(ctrl)
			keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).Return(apperror.ErrUnavailable)

			err := provideRealtimeReprovisioner(sub, keeper, "inst-1").Reprovision(t.Context())
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}
