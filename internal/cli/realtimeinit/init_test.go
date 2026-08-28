package realtimeinit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

var errBoom = xerrors.New("boom")

func TestTableNames(t *testing.T) {
	t.Parallel()

	names := TableNames(config.NewRealtimeConfig(config.MockConfigForTest(t)))

	assert.Equal(t, []string{"realtime_event_log_test", "realtime_stream_ticket_test", "realtime_instance_lease_test"}, names)
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("3 table を順に実在させる", func(t *testing.T) {
			t.Parallel()

			var names []string
			ensure := func(_ context.Context, table string) error {
				names = append(names, table)

				return nil
			}

			cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
			require.NoError(t, Run(t.Context(), cfg, ensure, logging.NewTestLogger(t)))
			assert.Equal(t, TableNames(cfg), names)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗した table 名を添えて止まり、残りは実行しない", func(t *testing.T) {
			t.Parallel()

			calls := 0
			ensure := func(_ context.Context, table string) error {
				calls++
				if table == "realtime_stream_ticket_test" {
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
