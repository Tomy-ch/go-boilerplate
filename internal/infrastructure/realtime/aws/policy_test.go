package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_queuePolicyDocument(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SNS からの sqs:SendMessage を topic の ARN 条件付きで許す 1 文の policy になる", func(t *testing.T) {
			t.Parallel()

			doc, err := queuePolicyDocument("arn:aws:sqs:r:1:q", "arn:aws:sns:r:1:t")
			require.NoError(t, err)
			assert.JSONEq(t, `{
				"Version":"2012-10-17",
				"Statement":[{
					"Sid":"AllowRealtimeTopic","Effect":"Allow",
					"Principal":{"Service":"sns.amazonaws.com"},
					"Action":"sqs:SendMessage","Resource":"arn:aws:sqs:r:1:q",
					"Condition":{"ArnEquals":{"aws:SourceArn":"arn:aws:sns:r:1:t"}}
				}]
			}`, doc)
		})
	})
}
