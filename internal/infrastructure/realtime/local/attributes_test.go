package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueueAttributes(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewQueueAttributes())
}

func Test_queueAttributes_Build(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("timings の 2 属性だけを返し、policy / redrive / 暗号化は含めない", func(t *testing.T) {
			t.Parallel()

			attrs, err := NewQueueAttributes().Build("arn:aws:sqs:r:1:q")
			require.NoError(t, err)

			assert.Equal(t, map[string]string{"VisibilityTimeout": "30", "ReceiveMessageWaitTimeSeconds": "20"}, attrs)
		})
	})
}
