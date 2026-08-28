package dynamodbclient

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/awsclient"
	"go-boilerplate/pkg/xerrors"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("endpoint と固定の retry 上限を持つクライアントを生成する", func(t *testing.T) {
			t.Parallel()

			c, err := New(t.Context(), Config{
				Endpoint: "http://localhost:8000", Region: "us-east-1", AccessKeyID: "k", SecretAccessKey: "s",
			})
			require.NoError(t, err)
			assert.Equal(t, "http://localhost:8000", aws.ToString(c.Options().BaseEndpoint))
			assert.Equal(t, MaxAttempts, c.Options().RetryMaxAttempts)
			assert.Equal(t, aws.RetryModeStandard, c.Options().RetryMode)
			assert.Equal(t, MaxAttempts, c.Options().Retryer.MaxAttempts())
		})

		t.Run("endpoint が空なら SDK 既定の解決に委ねる", func(t *testing.T) {
			t.Parallel()

			c, err := New(t.Context(), Config{Region: "us-east-1", AccessKeyID: "k", SecretAccessKey: "s"})
			require.NoError(t, err)
			assert.Nil(t, c.Options().BaseEndpoint)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("資格情報が片方だけなら生成に失敗する", func(t *testing.T) {
			t.Parallel()

			_, err := New(t.Context(), Config{Region: "us-east-1", AccessKeyID: "k"})
			require.ErrorIs(t, err, awsclient.ErrInvalidCredentials)
		})
	})
}

func TestIsConditionalCheckFailed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("条件式の不成立なら true", func(t *testing.T) {
			t.Parallel()

			err := xerrors.Wrap(&types.ConditionalCheckFailedException{Message: aws.String("failed")}, "put")
			assert.True(t, IsConditionalCheckFailed(err))
		})

		t.Run("別のエラーなら false", func(t *testing.T) {
			t.Parallel()

			assert.False(t, IsConditionalCheckFailed(&types.ResourceNotFoundException{}))
			assert.False(t, IsConditionalCheckFailed(nil))
		})
	})
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("context の取り消しは ErrCanceled になる", func(t *testing.T) {
			t.Parallel()

			err := Normalize(xerrors.Wrap(context.DeadlineExceeded, "call"), "query")
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Contains(t, err.Error(), "query")
		})

		t.Run("それ以外は ErrUnavailable になる", func(t *testing.T) {
			t.Parallel()

			err := Normalize(&types.ResourceNotFoundException{Message: aws.String("no table")}, "put")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Contains(t, err.Error(), "put")
		})
	})
}

func Test_isResourceInUse(t *testing.T) {
	t.Parallel()

	assert.True(t, isResourceInUse(xerrors.Wrap(&types.ResourceInUseException{}, "create")))
	assert.False(t, isResourceInUse(&types.ResourceNotFoundException{}))
	assert.False(t, isResourceInUse(nil))
}

// contractClient は、DynamoDB Local（または REALTIME_TEST_* の向け先）に繋ぐ contract test 用クライアントです。
func contractClient(t *testing.T) *dynamodb.Client {
	t.Helper()

	conn := config.NewRealtimeTestConnection(t)
	c, err := New(t.Context(), Config{
		Endpoint: conn.Endpoint, Region: conn.Region, AccessKeyID: conn.AccessKeyID, SecretAccessKey: conn.SecretAccessKey,
	})
	require.NoError(t, err)

	return c
}

func contractTableName(t *testing.T) string {
	t.Helper()

	b := make([]byte, 6)
	_, err := rand.Read(b)
	require.NoError(t, err)

	return "test_ensure_" + hex.EncodeToString(b)
}

func contractSpec(name, ttl string) TableSpec {
	return TableSpec{
		Name:         name,
		Attributes:   []types.AttributeDefinition{{AttributeName: aws.String("pk"), AttributeType: types.ScalarAttributeTypeS}},
		KeySchema:    []types.KeySchemaElement{{AttributeName: aws.String("pk"), KeyType: types.KeyTypeHash}},
		TTLAttribute: ttl,
	}
}

func dropTable(t *testing.T, c *dynamodb.Client, name string) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), tableWaitTimeout)
		defer cancel()
		_, _ = c.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(name)})
	})
}

func TestEnsureTable(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無い table を作り、2 回目も同じ状態のまま成功する", func(t *testing.T) {
			t.Parallel()

			c := contractClient(t)
			name := contractTableName(t)
			dropTable(t, c, name)

			require.NoError(t, EnsureTable(t.Context(), c, contractSpec(name, "expires_at")))
			require.NoError(t, EnsureTable(t.Context(), c, contractSpec(name, "expires_at")))

			desc, err := c.DescribeTable(t.Context(), &dynamodb.DescribeTableInput{TableName: aws.String(name)})
			require.NoError(t, err)
			assert.Equal(t, types.TableStatusActive, desc.Table.TableStatus)

			ttl, err := c.DescribeTimeToLive(t.Context(), &dynamodb.DescribeTimeToLiveInput{TableName: aws.String(name)})
			require.NoError(t, err)
			assert.Equal(t, "expires_at", aws.ToString(ttl.TimeToLiveDescription.AttributeName))
		})

		t.Run("TTL 属性が空なら TTL を設定しない", func(t *testing.T) {
			t.Parallel()

			c := contractClient(t)
			name := contractTableName(t)
			dropTable(t, c, name)

			require.NoError(t, EnsureTable(t.Context(), c, contractSpec(name, "")))

			ttl, err := c.DescribeTimeToLive(t.Context(), &dynamodb.DescribeTimeToLiveInput{TableName: aws.String(name)})
			require.NoError(t, err)
			assert.Equal(t, types.TimeToLiveStatusDisabled, ttl.TimeToLiveDescription.TimeToLiveStatus)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("store に繋がらなければ ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			c, err := New(t.Context(), Config{Endpoint: "http://127.0.0.1:1", Region: "us-east-1", AccessKeyID: "k", SecretAccessKey: "s"})
			require.NoError(t, err)

			err = EnsureTable(t.Context(), c, contractSpec("test_unreachable", ""))
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_ensureTimeToLive(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未設定なら有効にし、既に有効なら変えずに成功する", func(t *testing.T) {
			t.Parallel()

			c := contractClient(t)
			name := contractTableName(t)
			dropTable(t, c, name)
			require.NoError(t, EnsureTable(t.Context(), c, contractSpec(name, "")))

			require.NoError(t, ensureTimeToLive(t.Context(), c, name, "expires_at"))
			require.NoError(t, ensureTimeToLive(t.Context(), c, name, "expires_at"))

			ttl, err := c.DescribeTimeToLive(t.Context(), &dynamodb.DescribeTimeToLiveInput{TableName: aws.String(name)})
			require.NoError(t, err)
			assert.Equal(t, "expires_at", aws.ToString(ttl.TimeToLiveDescription.AttributeName))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			err := ensureTimeToLive(t.Context(), contractClient(t), "test_missing_"+contractTableName(t), "expires_at")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func TestStringAttr(t *testing.T) {
	t.Parallel()

	item := map[string]types.AttributeValue{"a": &types.AttributeValueMemberS{Value: "v"}, "n": &types.AttributeValueMemberN{Value: "1"}}
	assert.Equal(t, "v", StringAttr(item, "a"))
	assert.Empty(t, StringAttr(item, "n"), "S 以外は空")
	assert.Empty(t, StringAttr(item, "missing"))
}

func TestNumberAttr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("N を int64 に読む", func(t *testing.T) {
			t.Parallel()

			v, err := NumberAttr(map[string]types.AttributeValue{"n": &types.AttributeValueMemberN{Value: "42"}}, "n", "test")
			require.NoError(t, err)
			assert.Equal(t, int64(42), v)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("N でなければ ErrInternal", func(t *testing.T) {
			t.Parallel()

			_, err := NumberAttr(map[string]types.AttributeValue{"n": &types.AttributeValueMemberS{Value: "42"}}, "n", "test")
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Contains(t, err.Error(), "test item: n")
		})

		t.Run("整数に読めなければ ErrInternal", func(t *testing.T) {
			t.Parallel()

			_, err := NumberAttr(map[string]types.AttributeValue{"n": &types.AttributeValueMemberN{Value: "1.5"}}, "n", "test")
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
