package outbox

import (
	"testing"

	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_validateEmit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知のチャネルで順序指定が無ければ通す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateEmit(EmitInput{Channel: outboxbndry.ChannelHTTP}))
		})

		t.Run("順序キーと 1 以上の位置が揃っていれば通す", func(t *testing.T) {
			t.Parallel()

			err := validateEmit(EmitInput{
				Channel:          outboxbndry.ChannelRealtime,
				OrderingKey:      "thread-1",
				OrderingSequence: 1,
			})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("チャネル未指定は未知のチャネルとして拒否する", func(t *testing.T) {
			t.Parallel()

			err := validateEmit(EmitInput{})
			assert.ErrorIs(t, err, outboxbndry.ErrUnknownChannel)
		})

		t.Run("既知でないチャネルを拒否する", func(t *testing.T) {
			t.Parallel()

			err := validateEmit(EmitInput{Channel: outboxbndry.Channel("broker")})
			assert.ErrorIs(t, err, outboxbndry.ErrUnknownChannel)
		})

		t.Run("順序キーの無い位置指定を拒否する", func(t *testing.T) {
			t.Parallel()

			err := validateEmit(EmitInput{Channel: outboxbndry.ChannelHTTP, OrderingSequence: 1})
			assert.ErrorIs(t, err, ErrInvalidOrdering)
		})

		t.Run("位置の無い順序キー指定を拒否する", func(t *testing.T) {
			t.Parallel()

			err := validateEmit(EmitInput{Channel: outboxbndry.ChannelRealtime, OrderingKey: "thread-1"})
			assert.ErrorIs(t, err, ErrInvalidOrdering)
		})
	})
}
