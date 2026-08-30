package aws

import (
	"context"
	"testing"
	"time"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

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

func Test_withCallTimeout(t *testing.T) {
	t.Parallel()

	// middleware が握った deadline を覗くため、stack の続きを差し替えて ctx を取り出す。
	deadlineOf := func(t *testing.T, operation string) time.Duration {
		t.Helper()

		stack := smithymiddleware.NewStack("test", smithyhttp.NewStackRequest)
		require.NoError(t, withCallTimeout()(stack))

		var got time.Duration
		require.NoError(t, stack.Initialize.Add(
			smithymiddleware.InitializeMiddlewareFunc("capture",
				func(
					ctx context.Context, _ smithymiddleware.InitializeInput, _ smithymiddleware.InitializeHandler,
				) (smithymiddleware.InitializeOutput, smithymiddleware.Metadata, error) {
					deadline, ok := ctx.Deadline()
					require.True(t, ok, "呼び出しに deadline が設定されていない")
					got = time.Until(deadline).Round(time.Second)

					return smithymiddleware.InitializeOutput{}, smithymiddleware.Metadata{}, nil
				},
			),
			smithymiddleware.After,
		))

		ctx := awsmiddleware.SetOperationName(t.Context(), operation)
		_, _, err := smithymiddleware.DecorateHandler(
			smithymiddleware.HandlerFunc(func(context.Context, any) (any, smithymiddleware.Metadata, error) {
				return nil, smithymiddleware.Metadata{}, nil
			}),
			stack,
		).Handle(ctx, nil)
		require.NoError(t, err)

		return got
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("通常の呼び出しには CallTimeout を与える", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, CallTimeout, deadlineOf(t, "DeleteQueue"))
		})

		t.Run("long polling する受信には待ち時間を超える上限を与える", func(t *testing.T) {
			t.Parallel()

			// CallTimeout を当てると待ち切る前に落ちて loop が回らない。
			got := deadlineOf(t, receiveOperation)
			assert.Equal(t, receiveCallTimeout, got)
			assert.Greater(t, got, receiveWaitSeconds*time.Second)
		})
	})
}
