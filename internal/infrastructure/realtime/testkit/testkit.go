// Package testkit は、Realtime Delivery の fan-out の contract test が使う SNS / SQS クライアントと topic を提供します。
// 既定は共有インフラ / CI service の GoAWS で、config.NewRealtimeTestConnection の環境変数で本番 SNS / SQS へ
// 向け直しても同じテストが走ります。ARN は組み立てず API の戻り値から取ります（emulator の account は設定で変わる）。
package testkit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/realtime/aws"
)

const (
	// nameRandomBytes は、topic 名に付ける乱数の長さです。共有の emulator で複数 checkout のテストが
	// 同時に走っても衝突しない程度に十分です。
	nameRandomBytes = 6
	// deleteTimeout は、後片付けに与える上限です。
	deleteTimeout = 30 * time.Second
)

// NewTestClients は、contract test 用の SNS / SQS クライアントを返します。
func NewTestClients(t *testing.T) aws.Clients {
	t.Helper()

	conn := config.NewRealtimeTestConnection(t)
	c, err := aws.NewClients(t.Context(), aws.ClientConfig{
		Endpoint:        conn.PubSubEndpoint,
		Region:          conn.Region,
		AccessKeyID:     conn.AccessKeyID,
		SecretAccessKey: conn.SecretAccessKey,
	})
	require.NoError(t, err)

	return c
}

// Name は、テスト実行ごとに一意な名前（`test-<base>-<乱数>`）を返します。topic 名と queue prefix に使います。
func Name(t *testing.T, base string) string {
	t.Helper()

	b := make([]byte, nameRandomBytes)
	_, err := rand.Read(b)
	require.NoError(t, err)

	return "test-" + base + "-" + hex.EncodeToString(b)
}

// CreateTopic は、name の topic を作り、テスト終了時に削除します。戻り値は API が返した ARN です。
func CreateTopic(t *testing.T, c aws.Clients, name string) string {
	t.Helper()

	out, err := c.SNS.CreateTopic(t.Context(), &sns.CreateTopicInput{Name: awssdk.String(name)})
	require.NoError(t, err)

	arn := awssdk.ToString(out.TopicArn)
	require.NotEmpty(t, arn)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
		defer cancel()

		if _, err := c.SNS.DeleteTopic(ctx, &sns.DeleteTopicInput{TopicArn: awssdk.String(arn)}); err != nil {
			t.Logf("[testkit] topic %s の削除に失敗しました: %v", arn, err)
		}
	})

	return arn
}
