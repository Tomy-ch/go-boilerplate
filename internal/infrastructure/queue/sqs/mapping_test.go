package sqs

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/xerrors"
)

func Test_normalizeError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilを返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, normalizeError(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("context_CanceledはErrCanceledへ正規化する", func(t *testing.T) {
			t.Parallel()

			err := normalizeError(context.Canceled)

			require.ErrorIs(t, err, apperror.ErrCanceled)
			require.ErrorIs(t, err, context.Canceled) // 原因も保持する
		})

		t.Run("その他のbroker由来エラーはErrUnavailableへ正規化する", func(t *testing.T) {
			t.Parallel()

			cause := xerrors.New("throttled")
			err := normalizeError(cause)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.ErrorIs(t, err, cause)
		})
	})
}

func Test_toMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("属性_ReceiptHandle_ReceiveCount_MessageGroupIdを反映する", func(t *testing.T) {
			t.Parallel()

			got := toMessage(types.Message{
				MessageId:     aws.String("id1"),
				Body:          aws.String("hello"),
				ReceiptHandle: aws.String("rh1"),
				Attributes: map[string]string{
					string(types.MessageSystemAttributeNameApproximateReceiveCount): "3",
					string(types.MessageSystemAttributeNameMessageGroupId):          "grp",
				},
				MessageAttributes: map[string]types.MessageAttributeValue{
					"traceparent": {DataType: aws.String(attrDataTypeString), StringValue: aws.String("tp-val")},
				},
			})

			assert.Equal(t, "id1", got.ID)
			assert.Equal(t, []byte("hello"), got.Body)
			assert.Equal(t, 3, got.ReceiveCount)
			assert.Equal(t, "grp", got.PartitionKey)
			assert.Equal(t, "tp-val", got.Attributes["traceparent"])
			assert.Equal(t, "rh1", got.Attributes[worker.AttrReceiptHandle])
		})

		t.Run("StringValueがnilの属性とReceiptHandleがnilは予約キーを付与しない", func(t *testing.T) {
			t.Parallel()

			got := toMessage(types.Message{
				MessageId: aws.String("id1"),
				Body:      aws.String("hello"),
				MessageAttributes: map[string]types.MessageAttributeValue{
					"traceparent": {DataType: aws.String(attrDataTypeString), StringValue: nil},
				},
			})

			assert.NotContains(t, got.Attributes, "traceparent")
			assert.NotContains(t, got.Attributes, worker.AttrReceiptHandle)
		})

		t.Run("ApproximateReceiveCountが無い場合ReceiveCountは0になる", func(t *testing.T) {
			t.Parallel()

			got := toMessage(types.Message{
				MessageId: aws.String("id1"),
				Body:      aws.String("hello"),
			})

			assert.Equal(t, 0, got.ReceiveCount)
		})
	})
}

func Test_parseApproxCount(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数値文字列をint64へ変換する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, int64(42), parseApproxCount("42"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字は0を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, int64(0), parseApproxCount(""))
		})

		t.Run("parse不能な文字列は0を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, int64(0), parseApproxCount("not-a-number"))
		})
	})
}

// Test_statsProvider_queueDepth は、statsProvider.queueDepth を検証します。
func Test_statsProvider_queueDepth(t *testing.T) {
	t.Parallel()
	t.Skip(
		"statsProvider.queueDepth は stats_test.go の Test_statsProvider_QueueStats（visible/not_visible/delayed のparseとGetQueueAttributesエラー分岐）で網羅されている",
	)
}
