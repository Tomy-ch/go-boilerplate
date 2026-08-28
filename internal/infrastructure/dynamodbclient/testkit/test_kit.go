// Package testkit は、Realtime Delivery の contract test が使う DynamoDB クライアントと table 名を提供します。
// 既定は共有インフラ / CI service の DynamoDB Local で、config.NewRealtimeTestConnection の環境変数で
// 本番 DynamoDB へ向け直しても同じテストが走ります。
package testkit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
)

const (
	// tableNameRandomBytes は、table 名に付ける乱数の長さです。共有の DynamoDB Local で複数 checkout の
	// テストが同時に走っても衝突しない程度に十分です。
	tableNameRandomBytes = 6
	// deleteTimeout は、後片付けの DeleteTable に与える上限です。
	deleteTimeout = 30 * time.Second
)

// NewTestClient は、contract test 用の DynamoDB クライアントを返します。
func NewTestClient(t *testing.T) *dynamodb.Client {
	t.Helper()

	conn := config.NewRealtimeTestConnection(t)
	c, err := dynamodbclient.New(t.Context(), dynamodbclient.Config{
		Endpoint:        conn.Endpoint,
		Region:          conn.Region,
		AccessKeyID:     conn.AccessKeyID,
		SecretAccessKey: conn.SecretAccessKey,
	})
	require.NoError(t, err)

	return c
}

// TableName は、テスト実行ごとに一意な table 名（小文字、`test_<base>_<乱数>`）を返します。
func TableName(t *testing.T, base string) string {
	t.Helper()

	b := make([]byte, tableNameRandomBytes)
	_, err := rand.Read(b)
	require.NoError(t, err)

	return "test_" + base + "_" + hex.EncodeToString(b)
}

// DeleteOnCleanup は、テスト終了時に table を削除します。テストの ctx はその時点で終わっているため、
// 独立の ctx で削除します。
func DeleteOnCleanup(t *testing.T, c *dynamodb.Client, table string) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
		defer cancel()

		_, err := c.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(table)})
		if err != nil {
			t.Logf("[testkit] table %s の削除に失敗しました: %v", table, err)
		}
	})
}
