package realtimeinit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

var errBoom = xerrors.New("boom")

func TestSpecs(t *testing.T) {
	t.Parallel()

	specs := Specs(config.NewRealtimeConfig(config.MockConfigForTest(t)))

	require.Len(t, specs, 3)
	assert.Equal(t, "realtime_event_log_test", specs[0].Name)
	assert.Equal(t, "realtime_stream_ticket_test", specs[1].Name)
	assert.Equal(t, "realtime_instance_lease_test", specs[2].Name)
	assert.NotEmpty(t, specs[0].TTLAttribute, "EventLog は TTL を持つ")
	assert.Empty(t, specs[2].TTLAttribute, "InstanceLease は TTL を持たない")
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("3 table を順に実在させる", func(t *testing.T) {
			t.Parallel()

			var names []string
			ensure := func(_ context.Context, spec dynamodbclient.TableSpec) error {
				names = append(names, spec.Name)

				return nil
			}

			cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
			require.NoError(t, Run(t.Context(), cfg, ensure, logging.NewTestLogger(t)))
			assert.Equal(t, []string{"realtime_event_log_test", "realtime_stream_ticket_test", "realtime_instance_lease_test"}, names)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗した table 名を添えて止まり、残りは実行しない", func(t *testing.T) {
			t.Parallel()

			calls := 0
			ensure := func(_ context.Context, spec dynamodbclient.TableSpec) error {
				calls++
				if spec.Name == "realtime_stream_ticket_test" {
					return errBoom
				}

				return nil
			}

			cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
			err := Run(t.Context(), cfg, ensure, logging.NewTestLogger(t))
			require.ErrorIs(t, err, errBoom)
			assert.Contains(t, err.Error(), "realtime_stream_ticket_test")
			assert.Equal(t, 2, calls)
		})
	})
}
