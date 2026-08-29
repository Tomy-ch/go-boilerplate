package testkit

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/infrastructure/realtime/aws"
	"go-boilerplate/internal/infrastructure/realtime/local"
	"go-boilerplate/internal/observability"
)

func TestNewTestClients(t *testing.T) {
	t.Parallel()

	c := NewTestClients(t)
	assert.NotNil(t, c.SNS)
	assert.NotNil(t, c.SQS)
}

func TestName(t *testing.T) {
	t.Parallel()

	a := Name(t, "topic")
	b := Name(t, "topic")

	assert.Regexp(t, `^test-topic-[0-9a-f]{12}$`, a)
	assert.NotEqual(t, a, b, "実行ごとに一意")
}

func TestCreateTopic(t *testing.T) {
	t.Parallel()

	c := NewTestClients(t)

	// Cleanup は後に登録したものから走るため、ここで登録した検証は CreateTopic の削除の後に回る。
	var arn string
	t.Cleanup(func() {
		_, err := c.SNS.GetTopicAttributes(context.Background(), &sns.GetTopicAttributesInput{TopicArn: awssdk.String(arn)})
		assert.Error(t, err, "テスト終了時に topic は削除されている")
	})

	arn = CreateTopic(t, c, Name(t, "create"))

	got, err := c.SNS.GetTopicAttributes(t.Context(), &sns.GetTopicAttributesInput{TopicArn: awssdk.String(arn)})
	require.NoError(t, err)
	assert.Equal(t, arn, got.Attributes["TopicArn"])
}

func TestTeardownOnCleanup(t *testing.T) {
	t.Parallel()

	c := NewTestClients(t)
	arn := CreateTopic(t, c, Name(t, "teardown"))
	prefix := Name(t, "q")
	sub := aws.NewInstanceSubscription(
		c.SNS, c.SQS, aws.SubscriptionTarget{TopicARN: arn, QueuePrefix: prefix},
		local.NewQueueAttributes(), observability.NewNoopTracerFactory(t),
	)
	require.NoError(t, sub.Provision(t.Context(), "inst1"))

	// Cleanup は後に登録したものから走るため、ここで登録した検証は TeardownOnCleanup の片付けの後に回る。
	t.Cleanup(func() {
		out, err := c.SQS.ListQueues(context.Background(), &sqs.ListQueuesInput{QueueNamePrefix: awssdk.String(prefix)})
		require.NoError(t, err)
		assert.Empty(t, out.QueueUrls, "テスト終了時に instance の queue は片付けられている")
	})

	TeardownOnCleanup(t, sub)
}
