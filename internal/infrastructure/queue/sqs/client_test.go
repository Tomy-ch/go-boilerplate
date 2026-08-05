package sqs

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/observability"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("endpoint を指定すると解決先を上書きする", func(t *testing.T) {
			t.Parallel()

			const endpoint = "http://elasticmq:9324"

			got := NewClient(ClientConfig{Endpoint: endpoint, Region: "us-east-1"})

			require.NotNil(t, got)
			assert.Equal(t, endpoint, aws.ToString(got.Options().BaseEndpoint))
		})

		t.Run("endpoint が空なら SDK 既定の解決に委ねる", func(t *testing.T) {
			t.Parallel()

			got := NewClient(ClientConfig{Region: "us-east-1"})

			require.NotNil(t, got)
			assert.Nil(t, got.Options().BaseEndpoint)
		})

		t.Run("静的資格情報とリージョンを反映する", func(t *testing.T) {
			t.Parallel()

			got := NewClient(ClientConfig{
				Region:          "ap-northeast-1",
				AccessKeyID:     "dummy-key",
				SecretAccessKey: "dummy-secret",
			})

			require.NotNil(t, got)
			assert.Equal(t, "ap-northeast-1", got.Options().Region)
			creds, err := got.Options().Credentials.Retrieve(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "dummy-key", creds.AccessKeyID)
		})

		t.Run("渡した HTTPClient を SDK がそのまま保持する", func(t *testing.T) {
			t.Parallel()
			// 非 BuildableClient は SDK が差し替えないため、注入したインスタンスが同一のまま残る。
			// ここが nil に戻ると SDK 既定のトランスポートになり SSRF ガードを失う。
			outbound := observability.NewDisabledOutboundHTTPClient(true)

			got := NewClient(ClientConfig{Region: "us-east-1", HTTPClient: outbound})

			assert.Same(t, outbound, got.Options().HTTPClient)
		})
	})
}
