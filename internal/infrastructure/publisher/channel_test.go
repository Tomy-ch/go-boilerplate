package publisher_test

import (
	"testing"

	"go-boilerplate/internal/infrastructure/publisher"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"

	"github.com/stretchr/testify/require"
)

func TestVerifyChannel(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("http チャネルは配送できる", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, publisher.VerifyChannel(outboxbndry.ChannelHTTP))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("realtime チャネルは配送する実装がまだ無いため拒否する", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, publisher.VerifyChannel(outboxbndry.ChannelRealtime), publisher.ErrChannelUnsupported)
		})
	})
}
