package publisher

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("http スキームの URL を受理する", func(t *testing.T) {
			t.Parallel()

			const raw = "http://localhost:8080/events"
			got, err := parseEndpoint(raw)

			require.NoError(t, err)
			assert.Equal(t, Endpoint(raw), got)
		})

		t.Run("https スキームの URL を受理する", func(t *testing.T) {
			t.Parallel()

			const raw = "https://example.com/ingest"
			got, err := parseEndpoint(raw)

			require.NoError(t, err)
			assert.Equal(t, Endpoint(raw), got)
		})

		t.Run("ホストのみの URL を受理する", func(t *testing.T) {
			t.Parallel()

			const raw = "http://receiver"
			got, err := parseEndpoint(raw)

			require.NoError(t, err)
			assert.Equal(t, Endpoint(raw), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字は弾く", func(t *testing.T) {
			t.Parallel()

			got, err := parseEndpoint("")

			require.ErrorIs(t, err, ErrInvalidEndpoint)
			assert.Empty(t, got)
		})

		t.Run("スキーム無しは弾く", func(t *testing.T) {
			t.Parallel()

			got, err := parseEndpoint("example.com/events")

			require.ErrorIs(t, err, ErrInvalidEndpoint)
			assert.Empty(t, got)
		})

		t.Run("http/https 以外のスキームは弾く", func(t *testing.T) {
			t.Parallel()

			got, err := parseEndpoint("ftp://example.com")

			require.ErrorIs(t, err, ErrInvalidEndpoint)
			assert.Empty(t, got)
		})

		t.Run("ホスト欠落は弾く", func(t *testing.T) {
			t.Parallel()

			got, err := parseEndpoint("http://")

			require.ErrorIs(t, err, ErrInvalidEndpoint)
			assert.Empty(t, got)
		})

		t.Run("ホスト名が空(ポートのみ)は弾く", func(t *testing.T) {
			t.Parallel()

			got, err := parseEndpoint("http://:8080")

			require.ErrorIs(t, err, ErrInvalidEndpoint)
			assert.Empty(t, got)
		})

		t.Run("解析不能な URL は弾く", func(t *testing.T) {
			t.Parallel()

			got, err := parseEndpoint("http://[::1")

			require.ErrorIs(t, err, ErrInvalidEndpoint)
			assert.Empty(t, got)
		})
	})
}

func TestNewEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("config の endpoint が有効なら Endpoint を返す", func(t *testing.T) {
			t.Parallel()

			const raw = "http://localhost:8080/events"
			epCfg := config.NewEndpointConfig(config.MockConfigForTest(t))
			epCfg.SetEndpointOutbox(t, raw)

			got, err := NewEndpoint(epCfg)

			require.NoError(t, err)
			assert.Equal(t, Endpoint(raw), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("config の endpoint が未設定なら ErrInvalidEndpoint を返す", func(t *testing.T) {
			t.Parallel()

			epCfg := config.NewEndpointConfig(config.MockConfigForTest(t))

			got, err := NewEndpoint(epCfg)

			require.ErrorIs(t, err, ErrInvalidEndpoint)
			assert.Empty(t, got)
		})
	})
}
