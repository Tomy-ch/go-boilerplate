package sqs

import (
	"context"
	"testing"

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

func newStatsProvider(t *testing.T, api API, cfg Config) worker.QueueStatsProvider {
	t.Helper()
	return NewQueueStatsProvider(api, cfg, observability.NewNoopTracerFactory(t))
}

// attrs は、GetQueueAttributes の戻り値（visible / not_visible / delayed）を組み立てます。
func attrs(visible, notVisible, delayed string) *awssqs.GetQueueAttributesOutput {
	return &awssqs.GetQueueAttributesOutput{
		Attributes: map[string]string{
			string(types.QueueAttributeNameApproximateNumberOfMessages):           visible,
			string(types.QueueAttributeNameApproximateNumberOfMessagesNotVisible): notVisible,
			string(types.QueueAttributeNameApproximateNumberOfMessagesDelayed):    delayed,
		},
	}
}

func TestNewQueueStatsProvider(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API_Config_TracerFactory から QueueStatsProvider を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)

			p := NewQueueStatsProvider(api, Config{QueueURL: "q"}, observability.NewNoopTracerFactory(t))

			assert.NotNil(t, p)
		})
	})
}

func Test_statsProvider_QueueStats(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("source の visible_not_visible_delayed を parse する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.GetQueueAttributesInput, _ ...func(*awssqs.Options)) (*awssqs.GetQueueAttributesOutput, error) {
					assert.Equal(t, "q", aws.ToString(in.QueueUrl))
					return attrs("10", "3", "1"), nil
				},
			)

			stats, err := newStatsProvider(t, api, Config{QueueURL: "q"}).QueueStats(context.Background())

			require.NoError(t, err)
			assert.Equal(t, int64(10), stats.Source.Visible)
			assert.Equal(t, int64(3), stats.Source.InFlight)
			assert.Equal(t, int64(1), stats.Source.Delayed)
			assert.Nil(t, stats.DLQ)
		})

		t.Run("DLQURL があれば DLQ の滞留量も取得する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.GetQueueAttributesInput, _ ...func(*awssqs.Options)) (*awssqs.GetQueueAttributesOutput, error) {
					assert.Equal(t, "q", aws.ToString(in.QueueUrl))
					return attrs("10", "0", "0"), nil
				},
			)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in *awssqs.GetQueueAttributesInput, _ ...func(*awssqs.Options)) (*awssqs.GetQueueAttributesOutput, error) {
					assert.Equal(t, "dlq", aws.ToString(in.QueueUrl))
					return attrs("5", "2", "4"), nil
				},
			)

			stats, err := newStatsProvider(t, api, Config{QueueURL: "q", DLQURL: "dlq"}).QueueStats(context.Background())

			require.NoError(t, err)
			require.NotNil(t, stats.DLQ)
			assert.Equal(t, int64(5), stats.DLQ.Visible)
			assert.Equal(t, int64(2), stats.DLQ.InFlight)
			assert.Equal(t, int64(4), stats.DLQ.Delayed)
		})

		t.Run("DLQURL が空なら DLQ 取得をスキップする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			// GetQueueAttributes は source の 1 回だけ呼ばれる（DLQ では呼ばれない）。
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).Return(attrs("1", "0", "0"), nil).Times(1)

			stats, err := newStatsProvider(t, api, Config{QueueURL: "q"}).QueueStats(context.Background())

			require.NoError(t, err)
			assert.Nil(t, stats.DLQ)
		})

		t.Run("attribute 欠落は 0 件として扱う", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).Return(
				&awssqs.GetQueueAttributesOutput{Attributes: map[string]string{}}, nil)

			stats, err := newStatsProvider(t, api, Config{QueueURL: "q"}).QueueStats(context.Background())

			require.NoError(t, err)
			assert.Equal(t, int64(0), stats.Source.Visible)
			assert.Equal(t, int64(0), stats.Source.InFlight)
			assert.Equal(t, int64(0), stats.Source.Delayed)
		})

		t.Run("parse 不能な値は 0 件として扱う", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).Return(attrs("abc", "", "x"), nil)

			stats, err := newStatsProvider(t, api, Config{QueueURL: "q"}).QueueStats(context.Background())

			require.NoError(t, err)
			assert.Equal(t, int64(0), stats.Source.Visible)
			assert.Equal(t, int64(0), stats.Source.InFlight)
			assert.Equal(t, int64(0), stats.Source.Delayed)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("source の GetQueueAttributes エラーは ErrUnavailable に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

			stats, err := newStatsProvider(t, api, Config{QueueURL: "q"}).QueueStats(context.Background())

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, worker.QueueStats{}, stats)
		})

		t.Run("DLQ の GetQueueAttributes エラーは ErrUnavailable に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).Return(attrs("1", "0", "0"), nil)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

			stats, err := newStatsProvider(t, api, Config{QueueURL: "q", DLQURL: "dlq"}).QueueStats(context.Background())

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, worker.QueueStats{}, stats)
		})

		t.Run("context.Canceled は ErrCanceled に正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

			stats, err := newStatsProvider(t, api, Config{QueueURL: "q"}).QueueStats(context.Background())

			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Equal(t, worker.QueueStats{}, stats)
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

func Test_statsProvider_queueDepth(t *testing.T) {
	t.Parallel()
	t.Skip(
		"statsProvider.queueDepth は stats_test.go の Test_statsProvider_QueueStats（visible/not_visible/delayed のparseとGetQueueAttributesエラー分岐）で網羅されている",
	)
}
