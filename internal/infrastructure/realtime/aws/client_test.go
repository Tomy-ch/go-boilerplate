package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/infrastructure/awsclient"
)

func TestNewClients(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("静的資格情報と endpoint から SNS / SQS の両クライアントができる", func(t *testing.T) {
			t.Parallel()

			c, err := NewClients(t.Context(), ClientConfig{
				Endpoint: "http://localhost:4100", Region: "us-east-1", AccessKeyID: "k", SecretAccessKey: "s",
			})
			require.NoError(t, err)
			assert.Equal(t, "http://localhost:4100", awssdk.ToString(c.SNS.Options().BaseEndpoint))
			assert.Equal(t, "http://localhost:4100", awssdk.ToString(c.SQS.Options().BaseEndpoint))
			assert.Equal(t, "us-east-1", c.SNS.Options().Region)
		})

		t.Run("endpoint が空なら BaseEndpoint を設定しない（SDK 既定の解決）", func(t *testing.T) {
			t.Parallel()

			c, err := NewClients(t.Context(), ClientConfig{Region: "us-east-1", AccessKeyID: "k", SecretAccessKey: "s"})
			require.NoError(t, err)
			assert.Nil(t, c.SNS.Options().BaseEndpoint)
			assert.Nil(t, c.SQS.Options().BaseEndpoint)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("資格情報が片方だけなら ErrInvalidCredentials", func(t *testing.T) {
			t.Parallel()

			_, err := NewClients(t.Context(), ClientConfig{Region: "us-east-1", AccessKeyID: "k"})
			require.ErrorIs(t, err, awsclient.ErrInvalidCredentials)
		})
	})
}
