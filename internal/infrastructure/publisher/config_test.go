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

		cases := map[string]string{
			"http スキームの URL を受理する":  "http://localhost:8080/events",
			"https スキームの URL を受理する": "https://example.com/ingest",
			"ホストのみの URL を受理する":      "http://receiver",
		}
		for name, raw := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				got, err := parseEndpoint(raw)

				require.NoError(t, err)
				assert.Equal(t, Endpoint(raw), got)
			})
		}
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		cases := map[string]string{
			"空文字は弾く":                "",
			"スキーム無しは弾く":             "example.com/events",
			"http/https 以外のスキームは弾く": "ftp://example.com",
			"ホスト欠落は弾く":              "http://",
			"解析不能な URL は弾く":         "http://[::1",
		}
		for name, raw := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				got, err := parseEndpoint(raw)

				require.ErrorIs(t, err, ErrInvalidEndpoint)
				assert.Empty(t, got)
			})
		}
	})
}

func TestNewEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("config の endpoint が有効なら Endpoint を返す", func(t *testing.T) {
			t.Parallel()

			const raw = "http://localhost:8080/events"
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxEndpoint(t, raw)

			got, err := NewEndpoint(cfg)

			require.NoError(t, err)
			assert.Equal(t, Endpoint(raw), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("config の endpoint が未設定なら ErrInvalidEndpoint を返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))

			got, err := NewEndpoint(cfg)

			require.ErrorIs(t, err, ErrInvalidEndpoint)
			assert.Empty(t, got)
		})
	})
}
