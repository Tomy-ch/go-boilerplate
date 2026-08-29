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

	assert.Equal(
		t,
		[]string{"realtime_event_log_test", "realtime_stream_ticket_test", "realtime_instance_lease_test"},
		names,
	)
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

func TestTopicName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ARN の末尾要素を返す", func(t *testing.T) {
			t.Parallel()

			name, err := TopicName("arn:aws:sns:us-east-1:000000000000:realtime-fanout-local")
			require.NoError(t, err)
			assert.Equal(t, "realtime-fanout-local", name)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		for name, arn := range map[string]string{"空": "", "区切りが無い": "topic", "末尾が空": "arn:aws:sns:r:1:"} {
			t.Run(name+"なら ErrTopicARNInvalid", func(t *testing.T) {
				t.Parallel()

				_, err := TopicName(arn)
				require.ErrorIs(t, err, ErrTopicARNInvalid)
			})
		}
	})
}

func TestRunTopic(t *testing.T) {
	t.Parallel()

	const arn = "arn:aws:sns:us-east-1:000000000000:realtime-fanout-local"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ARN から導いた名前で topic を実在させ、返った ARN が設定と一致すれば成功", func(t *testing.T) {
			t.Parallel()

			var asked string
			ensure := func(_ context.Context, name string) (string, error) {
				asked = name

				return arn, nil
			}

			require.NoError(t, RunTopic(t.Context(), arn, ensure, logging.NewTestLogger(t)))
			assert.Equal(t, "realtime-fanout-local", asked)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ARN が不正なら topic を作らない", func(t *testing.T) {
			t.Parallel()

			ensure := func(context.Context, string) (string, error) {
				t.Fatal("called")

				return "", nil
			}

			require.ErrorIs(t, RunTopic(t.Context(), "", ensure, logging.NewTestLogger(t)), ErrTopicARNInvalid)
		})

		t.Run("作成の失敗は topic 名を添えて返す", func(t *testing.T) {
			t.Parallel()

			ensure := func(context.Context, string) (string, error) { return "", errBoom }

			err := RunTopic(t.Context(), arn, ensure, logging.NewTestLogger(t))
			require.ErrorIs(t, err, errBoom)
			assert.Contains(t, err.Error(), "realtime-fanout-local")
		})

		t.Run("返った ARN が設定と違えば ErrTopicARNMismatch", func(t *testing.T) {
			t.Parallel()

			ensure := func(context.Context, string) (string, error) {
				return "arn:aws:sns:us-east-1:100010001000:realtime-fanout-local", nil
			}

			err := RunTopic(t.Context(), arn, ensure, logging.NewTestLogger(t))
			require.ErrorIs(t, err, ErrTopicARNMismatch)
			assert.Contains(t, err.Error(), "100010001000")
		})
	})
}
