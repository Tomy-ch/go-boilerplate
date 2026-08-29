package aws

import (
	"testing"

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
			assert.NotNil(t, c.SNS)
			assert.NotNil(t, c.SQS)
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
