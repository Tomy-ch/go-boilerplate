package outbox_test

import (
	"testing"

	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseChannel(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("http を Channel へ変換する", func(t *testing.T) {
			t.Parallel()

			got, err := outboxbndry.ParseChannel("http")
			require.NoError(t, err)
			assert.Equal(t, outboxbndry.ChannelHTTP, got)
		})

		t.Run("realtime を Channel へ変換する", func(t *testing.T) {
			t.Parallel()

			got, err := outboxbndry.ParseChannel("realtime")
			require.NoError(t, err)
			assert.Equal(t, outboxbndry.ChannelRealtime, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字は既知でないチャネルとして拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := outboxbndry.ParseChannel("")
			require.ErrorIs(t, err, outboxbndry.ErrUnknownChannel)
		})

		t.Run("既知でない値は拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := outboxbndry.ParseChannel("broker")
			require.ErrorIs(t, err, outboxbndry.ErrUnknownChannel)
		})

		t.Run("大文字違いは別物として拒否する", func(t *testing.T) {
			t.Parallel()

			// DB の CHECK 制約は小文字の値だけを許すため、正規化して受け入れない。
			_, err := outboxbndry.ParseChannel("HTTP")
			require.ErrorIs(t, err, outboxbndry.ErrUnknownChannel)
		})
	})
}

func TestChannel_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("チャネル名をそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "realtime", outboxbndry.ChannelRealtime.String())
		})
	})
}
