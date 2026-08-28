package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

// realtimeDeps は、realtime module の graph 検証に要る下位モジュール群です（clock は infrastructure 側の module）。
func realtimeDeps() []fx.Option {
	return append(commonDeps(), clockModule(), realtimeModule())
}

func Test_realtimeModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	validateGraph(t, realtimeDeps()...)
}

func Test_realtimeModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("3 つの store と機構側 usecase を提供する", func(t *testing.T) {
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

			validateGraph(t, append(realtimeDeps(), fx.Populate(&log, &tickets, &leases, &secrets, &cursor, &issuer, &verifier))...)
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
				config.NewRealtimeConfig(mock), config.NewEndpointConfig(mock), observability.NewDisabledOutboundHTTPClient(true))
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
	assert.NotNil(t, provideCursorValidator(log, mock_clock.NewMockClock(gomock.NewController(t)), observability.NewNoopTracerFactory(t)))
}

func Test_provideTicketIssuer(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	store := provideStreamTicketStore(testkit.NewTestClient(t), cfg, observability.NewNoopTracerFactory(t))
	assert.NotNil(t, provideTicketIssuer(store, provideRealtimeSecretGenerator(), mock_clock.NewMockClock(gomock.NewController(t)), observability.NewNoopTracerFactory(t)))
}

func Test_provideTicketVerifier(t *testing.T) {
	t.Parallel()

	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	store := provideStreamTicketStore(testkit.NewTestClient(t), cfg, observability.NewNoopTracerFactory(t))
	assert.NotNil(t, provideTicketVerifier(store, mock_clock.NewMockClock(gomock.NewController(t)), observability.NewNoopTracerFactory(t)))
}
