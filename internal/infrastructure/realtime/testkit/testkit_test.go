package testkit

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
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
	arn := CreateTopic(t, c, Name(t, "create"))

	got, err := c.SNS.GetTopicAttributes(t.Context(), &sns.GetTopicAttributesInput{TopicArn: awssdk.String(arn)})
	require.NoError(t, err)
	assert.Equal(t, arn, got.Attributes["TopicArn"])
}

func TestTeardownOnCleanup(t *testing.T) {
	t.Parallel()

	c := NewTestClients(t)
	arn := CreateTopic(t, c, Name(t, "teardown"))
	sub := aws.NewInstanceSubscription(
		c.SNS, c.SQS, aws.SubscriptionTarget{TopicARN: arn, QueuePrefix: Name(t, "q")},
		local.NewQueueAttributes(), observability.NewNoopTracerFactory(t),
	)
	require.NoError(t, sub.Provision(t.Context(), "inst1"))

	// Cleanup で片付けられることは、この test の終了後に queue が残らないことで示される（片付け失敗は t.Logf に出る）。
	TeardownOnCleanup(t, sub)
}
