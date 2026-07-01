package sqs

import (
	"context"
	"go/constant"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	mock_sqs "go-boilerplate/internal/infrastructure/queue/sqs/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/worker"
)

func newConsumer(t *testing.T, api API) worker.Consumer {
	t.Helper()
	return NewConsumer(api, Config{QueueURL: "q", MaxMessages: 10}, observability.NewNoopTracerFactory(t))
}

func newConsumerWithConfig(t *testing.T, api API, cfg Config) worker.Consumer {
	t.Helper()
	return NewConsumer(api, cfg, observability.NewNoopTracerFactory(t))
}

func TestNewConsumer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API_Config_TracerFactory から Consumer を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)

			c := NewConsumer(api, Config{QueueURL: "q"}, observability.NewNoopTracerFactory(t))

			assert.NotNil(t, c)
		})
	})
}

func TestNewDeadLetter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API_DLQURL_TracerFactory から FailureHandler を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)

			dl := NewDeadLetter(api, "dlq", observability.NewNoopTracerFactory(t))

			assert.NotNil(t, dl)
		})
	})
}

func Test_Consumer_Receive(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SQS メッセージを broker 非依存の Message へ正規化する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).Return(&awssqs.ReceiveMessageOutput{
				Messages: []types.Message{{
					MessageId:     aws.String("id1"),
					Body:          aws.String("hello"),
					ReceiptHandle: aws.String("rh1"),
					Attributes: map[string]string{
						string(types.MessageSystemAttributeNameApproximateReceiveCount): "3",
						string(types.MessageSystemAttributeNameMessageGroupId):          "grp",
					},
					MessageAttributes: map[string]types.MessageAttributeValue{
						"traceparent": {DataType: aws.String(constant.String.String()), StringValue: aws.String("tp-val")},
					},
				}},
			}, nil)

			msgs, err := newConsumer(t, api).Receive(context.Background(), 5)

			require.NoError(t, err)
			require.Len(t, msgs, 1)
			got := msgs[0]
			assert.Equal(t, "id1", got.ID)
			assert.Equal(t, "hello", string(got.Body))
			assert.Equal(t, 3, got.ReceiveCount)
			assert.Equal(t, "grp", got.PartitionKey)
			assert.Equal(t, "tp-val", got.Attributes["traceparent"])
			assert.Equal(t, "rh1", got.Attributes[worker.AttrReceiptHandle])
		})

		t.Run("StringValue が nil の属性と ReceiptHandle が nil のメッセージは予約キーを付与しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).Return(&awssqs.ReceiveMessageOutput{
				Messages: []types.Message{{
					MessageId: aws.String("id1"),
					Body:      aws.String("hello"),
					MessageAttributes: map[string]types.MessageAttributeValue{
						"traceparent": {DataType: aws.String(constant.String.String()), StringValue: nil},
					},
				}},
			}, nil)

			msgs, err := newConsumer(t, api).Receive(context.Background(), 5)

			require.NoError(t, err)
			require.Len(t, msgs, 1)
			assert.NotContains(t, msgs[0].Attributes, "traceparent")
			assert.NotContains(t, msgs[0].Attributes, worker.AttrReceiptHandle)
		})

		t.Run("ApproximateReceiveCount 属性が無い場合 ReceiveCount は 0 になる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).Return(&awssqs.ReceiveMessageOutput{
				Messages: []types.Message{{
					MessageId:     aws.String("id1"),
					Body:          aws.String("hello"),
					ReceiptHandle: aws.String("rh1"),
					Attributes: map[string]string{
						string(types.MessageSystemAttributeNameMessageGroupId): "grp",
					},
				}},
			}, nil)

			msgs, err := newConsumer(t, api).Receive(context.Background(), 5)

			require.NoError(t, err)
			require.Len(t, msgs, 1)
			assert.Equal(t, 0, msgs[0].ReceiveCount)
		})

		t.Run("メッセージが0件の場合は空スライスを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).Return(&awssqs.ReceiveMessageOutput{Messages: nil}, nil)

			msgs, err := newConsumer(t, api).Receive(context.Background(), 5)

			require.NoError(t, err)
			assert.Empty(t, msgs)
		})

		t.Run("MaxMessages 設定が要求件数より小さい場合は設定値に丸める", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.ReceiveMessageInput, _ ...func(*awssqs.Options)) (*awssqs.ReceiveMessageOutput, error) {
					assert.Equal(t, int32(3), in.MaxNumberOfMessages)
					return &awssqs.ReceiveMessageOutput{}, nil
				},
			)

			cfg := Config{QueueURL: "q", MaxMessages: 3}
			msgs, err := newConsumerWithConfig(t, api, cfg).Receive(context.Background(), 5)

			require.NoError(t, err)
			assert.Empty(t, msgs)
		})

		t.Run("要求件数が SQS 上限を超える場合は10に丸める", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.ReceiveMessageInput, _ ...func(*awssqs.Options)) (*awssqs.ReceiveMessageOutput, error) {
					assert.Equal(t, int32(10), in.MaxNumberOfMessages)
					return &awssqs.ReceiveMessageOutput{}, nil
				},
			)

			cfg := Config{QueueURL: "q"}
			msgs, err := newConsumerWithConfig(t, api, cfg).Receive(context.Background(), 50)

			require.NoError(t, err)
			assert.Empty(t, msgs)
		})

		t.Run("要求件数が1未満の場合は1に補正する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.ReceiveMessageInput, _ ...func(*awssqs.Options)) (*awssqs.ReceiveMessageOutput, error) {
					assert.Equal(t, int32(1), in.MaxNumberOfMessages)
					return &awssqs.ReceiveMessageOutput{}, nil
				},
			)

			cfg := Config{QueueURL: "q"}
			msgs, err := newConsumerWithConfig(t, api, cfg).Receive(context.Background(), 0)

			require.NoError(t, err)
			assert.Empty(t, msgs)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API エラーは ErrUnavailable に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

			msgs, err := newConsumer(t, api).Receive(context.Background(), 5)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, msgs)
		})

		t.Run("context.Canceled は ErrCanceled に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

			msgs, err := newConsumer(t, api).Receive(context.Background(), 5)

			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, msgs)
		})
	})
}

func Test_Consumer_Ack(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("予約キーの receipt handle で DeleteMessage する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().DeleteMessage(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.DeleteMessageInput, _ ...func(*awssqs.Options)) (*awssqs.DeleteMessageOutput, error) {
					assert.Equal(t, "q", aws.ToString(in.QueueUrl))
					assert.Equal(t, "rh1", aws.ToString(in.ReceiptHandle))
					return &awssqs.DeleteMessageOutput{}, nil
				},
			)

			err := newConsumer(t, api).Ack(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			})

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API エラーは ErrUnavailable に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().DeleteMessage(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

			err := newConsumer(t, api).Ack(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("context.Canceled は ErrCanceled に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().DeleteMessage(gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

			err := newConsumer(t, api).Ack(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			})

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_Consumer_Nack(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("可視性を 0 にして即時再配送する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.ChangeMessageVisibilityInput, _ ...func(*awssqs.Options)) (*awssqs.ChangeMessageVisibilityOutput, error) {
					assert.Equal(t, int32(0), in.VisibilityTimeout)
					assert.Equal(t, "rh1", aws.ToString(in.ReceiptHandle))
					return &awssqs.ChangeMessageVisibilityOutput{}, nil
				},
			)

			err := newConsumer(t, api).Nack(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			})

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API エラーは ErrUnavailable に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

			err := newConsumer(t, api).Nack(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("context.Canceled は ErrCanceled に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

			err := newConsumer(t, api).Nack(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			})

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_Consumer_NackWithBackoff(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("可視性を遅延秒数に設定して遅延再配送する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.ChangeMessageVisibilityInput, _ ...func(*awssqs.Options)) (*awssqs.ChangeMessageVisibilityOutput, error) {
					assert.Equal(t, int32(5), in.VisibilityTimeout)
					assert.Equal(t, "rh1", aws.ToString(in.ReceiptHandle))
					return &awssqs.ChangeMessageVisibilityOutput{}, nil
				},
			)

			err := newConsumer(t, api).NackWithBackoff(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			}, 5*time.Second)

			require.NoError(t, err)
		})

		t.Run("遅延が0以下の場合は即時再配送する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.ChangeMessageVisibilityInput, _ ...func(*awssqs.Options)) (*awssqs.ChangeMessageVisibilityOutput, error) {
					assert.Equal(t, int32(0), in.VisibilityTimeout)
					return &awssqs.ChangeMessageVisibilityOutput{}, nil
				},
			)

			err := newConsumer(t, api).NackWithBackoff(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			}, 0)

			require.NoError(t, err)
		})

		t.Run("サブ秒の遅延は切り上げて可視性を1秒にする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.ChangeMessageVisibilityInput, _ ...func(*awssqs.Options)) (*awssqs.ChangeMessageVisibilityOutput, error) {
					assert.Equal(t, int32(1), in.VisibilityTimeout)
					return &awssqs.ChangeMessageVisibilityOutput{}, nil
				},
			)

			err := newConsumer(t, api).NackWithBackoff(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			}, 500*time.Millisecond)

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API エラーは ErrUnavailable に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

			err := newConsumer(t, api).NackWithBackoff(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			}, 5*time.Second)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("context.Canceled は ErrCanceled に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

			err := newConsumer(t, api).NackWithBackoff(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			}, 5*time.Second)

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_Consumer_Extend(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定時間で可視性を延長する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.ChangeMessageVisibilityInput, _ ...func(*awssqs.Options)) (*awssqs.ChangeMessageVisibilityOutput, error) {
					assert.Equal(t, int32(30), in.VisibilityTimeout)
					return &awssqs.ChangeMessageVisibilityOutput{}, nil
				},
			)

			err := newConsumer(t, api).Extend(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			}, 30*time.Second)

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API エラーは ErrUnavailable に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

			err := newConsumer(t, api).Extend(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			}, 30*time.Second)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("context.Canceled は ErrCanceled に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

			err := newConsumer(t, api).Extend(context.Background(), worker.Message{
				Attributes: map[string]string{worker.AttrReceiptHandle: "rh1"},
			}, 30*time.Second)

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_DeadLetter_Fail(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DLQ へ本文と失敗理由 permanent を SendMessage する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().SendMessage(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.SendMessageInput, _ ...func(*awssqs.Options)) (*awssqs.SendMessageOutput, error) {
					assert.Equal(t, "dlq", aws.ToString(in.QueueUrl))
					assert.Equal(t, "body", aws.ToString(in.MessageBody))
					assert.Equal(t, "permanent", aws.ToString(in.MessageAttributes["failure_reason"].StringValue))
					return &awssqs.SendMessageOutput{}, nil
				},
			)

			dl := NewDeadLetter(api, "dlq", observability.NewNoopTracerFactory(t))
			err := dl.Fail(context.Background(), worker.Message{Body: []byte("body")}, assert.AnError)

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API エラーは ErrUnavailable に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

			dl := NewDeadLetter(api, "dlq", observability.NewNoopTracerFactory(t))
			err := dl.Fail(context.Background(), worker.Message{Body: []byte("body")}, assert.AnError)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("context.Canceled は ErrCanceled に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

			dl := NewDeadLetter(api, "dlq", observability.NewNoopTracerFactory(t))
			err := dl.Fail(context.Background(), worker.Message{Body: []byte("body")}, assert.AnError)

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
