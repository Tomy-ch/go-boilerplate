package outbox_test

import (
	"context"
	"testing"

	outboxcli "go-boilerplate/internal/cli/outbox"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunReplayWith(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("messageID が空なら nil を渡して全 dead を対象にする", func(t *testing.T) {
			t.Parallel()

			var got *uuid.UUID
			gotCalled := false
			replay := func(_ context.Context, id *uuid.UUID) (int64, error) {
				got, gotCalled = id, true
				return 3, nil
			}

			count, err := outboxcli.RunReplayWith(context.Background(), "", replay)
			require.NoError(t, err)
			assert.Equal(t, int64(3), count)
			assert.True(t, gotCalled)
			assert.Nil(t, got)
		})

		t.Run("messageID 指定時は parse して渡す", func(t *testing.T) {
			t.Parallel()

			want := uuid.NewTestFromSalt(t, "dead")
			var got *uuid.UUID
			replay := func(_ context.Context, id *uuid.UUID) (int64, error) {
				got = id
				return 1, nil
			}

			count, err := outboxcli.RunReplayWith(context.Background(), want.String(), replay)
			require.NoError(t, err)
			assert.Equal(t, int64(1), count)
			require.NotNil(t, got)
			assert.Equal(t, want, *got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正な messageID は replay を呼ばずエラーを返す", func(t *testing.T) {
			t.Parallel()

			called := false
			replay := func(context.Context, *uuid.UUID) (int64, error) {
				called = true
				return 0, nil
			}

			_, err := outboxcli.RunReplayWith(context.Background(), "not-a-uuid", replay)
			require.Error(t, err)
			assert.False(t, called)
		})
	})
}
