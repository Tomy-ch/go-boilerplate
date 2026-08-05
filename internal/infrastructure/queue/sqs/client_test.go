package sqs

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/infrastructure/awsclient"
	"go-boilerplate/internal/observability"
)

// staticClientConfig は、資格情報だけをダミーの静的値で埋めた設定を返します。
// 空のままでは SDK 既定の credential chain へ委ねられ、実行マシンの共有プロファイルや
// IAM ロールの有無で結果が変わるため、資格情報の解決を検証対象にしないケースで固定します。
func staticClientConfig(cfg ClientConfig) ClientConfig {
	cfg.AccessKeyID = "dummy-key"
	cfg.SecretAccessKey = "dummy-secret"
	return cfg
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("endpoint を指定すると解決先を上書きする", func(t *testing.T) {
			t.Parallel()

			const endpoint = "http://elasticmq:9324"

			got, err := NewClient(t.Context(), staticClientConfig(ClientConfig{
				Endpoint: endpoint,
				Region:   "us-east-1",
			}))

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, endpoint, aws.ToString(got.Options().BaseEndpoint))
		})

		t.Run("endpoint が空なら SDK 既定の解決に委ねる", func(t *testing.T) {
			t.Parallel()

			got, err := NewClient(t.Context(), staticClientConfig(ClientConfig{Region: "us-east-1"}))

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Nil(t, got.Options().BaseEndpoint)
		})

		t.Run("静的資格情報とリージョンを反映する", func(t *testing.T) {
			t.Parallel()

			got, err := NewClient(t.Context(), ClientConfig{
				Region:          "ap-northeast-1",
				AccessKeyID:     "dummy-key",
				SecretAccessKey: "dummy-secret",
			})

			require.NoError(t, err)
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

			got, err := NewClient(t.Context(), staticClientConfig(ClientConfig{
				Region:     "us-east-1",
				HTTPClient: outbound,
			}))

			require.NoError(t, err)
			assert.Same(t, outbound, got.Options().HTTPClient)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("資格情報の解決に失敗したらクライアントを返さない", func(t *testing.T) {
			t.Parallel()

			got, err := NewClient(t.Context(), ClientConfig{Region: "us-east-1", AccessKeyID: "dummy-key"})

			require.ErrorIs(t, err, awsclient.ErrInvalidCredentials)
			assert.Nil(t, got)
		})
	})
}
