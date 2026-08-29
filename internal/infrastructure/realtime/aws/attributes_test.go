package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueueAttributes(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewQueueAttributes(QueueAttributesInput{TopicARN: "arn:topic"}))
}

func Test_queueAttributes_Build(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DLQ 無しなら policy / 暗号化 / timings の 4 属性", func(t *testing.T) {
			t.Parallel()

			attrs, err := NewQueueAttributes(QueueAttributesInput{TopicARN: "arn:aws:sns:r:1:t"}).Build("arn:aws:sqs:r:1:q")
			require.NoError(t, err)

			assert.Len(t, attrs, 4)
			assert.Equal(t, "30", attrs["VisibilityTimeout"])
			assert.Equal(t, "20", attrs["ReceiveMessageWaitTimeSeconds"])
			assert.Equal(t, "true", attrs["SqsManagedSseEnabled"])
			assert.Contains(t, attrs["Policy"], `"aws:SourceArn":"arn:aws:sns:r:1:t"`)
			assert.Contains(t, attrs["Policy"], `"Resource":"arn:aws:sqs:r:1:q"`)
			assert.NotContains(t, attrs, "RedrivePolicy")
		})

		t.Run("DLQ ありなら RedrivePolicy（maxReceiveCount=5）が加わる", func(t *testing.T) {
			t.Parallel()

			attrs, err := NewQueueAttributes(
				QueueAttributesInput{TopicARN: "arn:aws:sns:r:1:t", DLQARN: "arn:aws:sqs:r:1:dlq"},
			).Build("arn:aws:sqs:r:1:q")
			require.NoError(t, err)

			assert.Len(t, attrs, 5)
			assert.JSONEq(
				t,
				`{"deadLetterTargetArn":"arn:aws:sqs:r:1:dlq","maxReceiveCount":"5"}`,
				attrs["RedrivePolicy"],
			)
		})
	})
}

func TestTimingAttributes(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		map[string]string{"VisibilityTimeout": "30", "ReceiveMessageWaitTimeSeconds": "20"},
		TimingAttributes(),
	)
}

func Test_redrivePolicyDocument(t *testing.T) {
	t.Parallel()

	doc, err := redrivePolicyDocument("arn:dlq")
	require.NoError(t, err)
	assert.JSONEq(t, `{"deadLetterTargetArn":"arn:dlq","maxReceiveCount":"5"}`, doc)
}
