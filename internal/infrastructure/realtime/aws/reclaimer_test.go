package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	mock_aws "go-boilerplate/internal/infrastructure/realtime/aws/mock"
	"go-boilerplate/internal/observability"
)

// orphanQueueName は、テストの instance 識別子から作られる queue の名前です。
const orphanQueueName = "realtime-test-inst-1"

func newReclaimer(t *testing.T) (*reclaimer, subscriptionMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := subscriptionMocks{sns: mock_aws.NewMockSNSAPI(ctrl), sqs: mock_aws.NewMockSQSAPI(ctrl)}
	r, ok := NewOrphanReclaimer(
		m.sns, m.sqs, SubscriptionTarget{TopicARN: testTopicARN, QueuePrefix: "realtime-test"},
		observability.NewNoopTracerFactory(t),
	).(*reclaimer)
	require.True(t, ok)

	return r, m
}

// subscriptionPage は、ListSubscriptionsByTopic の 1 ページ分の応答を作ります。
func subscriptionPage(next string, endpoints ...string) *sns.ListSubscriptionsByTopicOutput {
	out := &sns.ListSubscriptionsByTopicOutput{}
	for i, e := range endpoints {
		out.Subscriptions = append(out.Subscriptions, snstypes.Subscription{
			Endpoint:        awssdk.String(e),
			SubscriptionArn: awssdk.String(testTopicARN + ":sub-" + string(rune('a'+i))),
		})
	}

	if next != "" {
		out.NextToken = awssdk.String(next)
	}

	return out
}

func TestNewOrphanReclaimer(t *testing.T) {
	t.Parallel()

	r, _ := newReclaimer(t)
	assert.NotNil(t, r)
}

func Test_reclaimer_Reclaim(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録を解除してから受信先を削除する", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			gomock.InOrder(
				m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).
					Return(subscriptionPage("", testQueueARN), nil),
				m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).Return(&sns.UnsubscribeOutput{}, nil),
				m.sqs.EXPECT().GetQueueUrl(gomock.Any(), gomock.Any()).
					Return(&sqs.GetQueueUrlOutput{QueueUrl: awssdk.String(testQueueURL)}, nil),
				m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil),
			)

			require.NoError(t, r.Reclaim(t.Context(), "inst-1"))
		})

		t.Run("QueuePrefix を変えた後でも、登録が持つ実際の名前から旧世代の残骸へ届く", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			// 旧 prefix で作られた受信先。設定から導ける名前（realtime-test-inst-1）とは一致しない。
			oldARN := "arn:aws:sqs:us-east-1:000000000000:realtime-old-inst-1"
			m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).
				Return(subscriptionPage("", oldARN), nil)
			m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).Return(&sns.UnsubscribeOutput{}, nil)

			// 旧名と現設定の名前の両方を試す。旧名は実在し、現設定の名前は実在しない。
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), &sqs.GetQueueUrlInput{
				QueueName: awssdk.String("realtime-old-inst-1"),
			}).Return(&sqs.GetQueueUrlOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil)
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), &sqs.GetQueueUrlInput{
				QueueName: awssdk.String(orphanQueueName),
			}).Return(nil, &sqstypes.QueueDoesNotExist{})

			require.NoError(t, r.Reclaim(t.Context(), "inst-1"))
		})

	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("片方が失敗しても残りを試み、失敗をまとめて返す", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).
				Return(subscriptionPage("", testQueueARN), nil)
			m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueUrlOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil)

			require.ErrorIs(t, r.Reclaim(t.Context(), "inst-1"), apperror.ErrUnavailable)
		})

		t.Run("登録の一覧が引けなくても、設定から導ける受信先の削除は試みる", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), &sqs.GetQueueUrlInput{
				QueueName: awssdk.String(orphanQueueName),
			}).Return(&sqs.GetQueueUrlOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil)

			require.ErrorIs(t, r.Reclaim(t.Context(), "inst-1"), apperror.ErrUnavailable)
		})

		t.Run("queue 名の規則を満たさない識別子は substrate を呼ばずに止まる", func(t *testing.T) {
			t.Parallel()

			r, _ := newReclaimer(t)

			require.ErrorIs(t, r.Reclaim(t.Context(), "inst/1"), ErrInvalidQueueName)
		})
	})
}

func Test_reclaimer_unsubscribeAll(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象の queue を指す登録だけを解除する", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			other := "arn:aws:sqs:us-east-1:000000000000:realtime-test-inst-2"
			m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).
				Return(subscriptionPage("", other, testQueueARN), nil)
			m.sns.EXPECT().Unsubscribe(gomock.Any(), &sns.UnsubscribeInput{
				SubscriptionArn: awssdk.String(testTopicARN + ":sub-b"),
			}).Return(&sns.UnsubscribeOutput{}, nil)

			found, err := r.unsubscribeAll(t.Context(), "inst-1")
			require.NoError(t, err)
			// 別 instance の受信先が混ざると、Reclaim が生きている instance の queue を消しにいく。
			assert.Equal(t, []string{orphanQueueName}, found)
		})

		t.Run("次のページも辿る", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			gomock.InOrder(
				m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), &sns.ListSubscriptionsByTopicInput{
					TopicArn: awssdk.String(testTopicARN),
				}).Return(subscriptionPage("page-2"), nil),
				m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), &sns.ListSubscriptionsByTopicInput{
					TopicArn: awssdk.String(testTopicARN), NextToken: awssdk.String("page-2"),
				}).Return(subscriptionPage("", testQueueARN), nil),
			)
			m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).Return(&sns.UnsubscribeOutput{}, nil)

			found, err := r.unsubscribeAll(t.Context(), "inst-1")
			require.NoError(t, err)
			assert.Equal(t, []string{orphanQueueName}, found)
		})

		t.Run("空の NextToken も終端として扱う", func(t *testing.T) {
			t.Parallel()

			// GoAWS は続きが無いとき nil ではなく空文字への非 nil ポインタを返す。nil だけを終端と
			// 見なすと同じページを取り続けて戻らない（実基盤で無限ループを踏んだ）。
			r, m := newReclaimer(t)
			empty := &sns.ListSubscriptionsByTopicOutput{NextToken: awssdk.String("")}
			m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).Return(empty, nil).Times(1)

			found, err := r.unsubscribeAll(t.Context(), "inst-1")
			require.NoError(t, err)
			assert.Empty(t, found)
		})

		t.Run("確認待ちの登録は解除を試みない", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).Return(
				&sns.ListSubscriptionsByTopicOutput{Subscriptions: []snstypes.Subscription{{
					Endpoint:        awssdk.String(testQueueARN),
					SubscriptionArn: awssdk.String("PendingConfirmation"),
				}}}, nil)

			found, err := r.unsubscribeAll(t.Context(), "inst-1")
			require.NoError(t, err)
			// 解除できなくても、削除の候補には載せる（載せないと孤児化した受信先が永久に残る）。
			assert.Equal(t, []string{orphanQueueName}, found)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧の取得に失敗したら解除を試みない", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			found, err := r.unsubscribeAll(t.Context(), "inst-1")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Empty(t, found)
		})

		t.Run("解除の失敗は残りを試みてからまとめて返す", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).
				Return(subscriptionPage("", testQueueARN, testQueueARN), nil)
			m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).Return(nil, errSubstrate).Times(2)

			found, err := r.unsubscribeAll(t.Context(), "inst-1")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, []string{orphanQueueName, orphanQueueName}, found)
		})

		t.Run("途中のページで一覧が失敗しても、それまでの失敗と見つけた受信先を返す", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			gomock.InOrder(
				m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).
					Return(subscriptionPage("page-2", testQueueARN), nil),
				m.sns.EXPECT().ListSubscriptionsByTopic(gomock.Any(), gomock.Any()).Return(nil, errSubstrate),
			)
			m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			found, err := r.unsubscribeAll(t.Context(), "inst-1")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			// 先行ページの解除失敗を捨てず、見つけた受信先も返す（どちらも回収の続行に要る）。
			assert.ErrorIs(t, err, errSubstrate)
			assert.Equal(t, []string{orphanQueueName}, found)
		})
	})
}

func Test_reclaimer_deleteQueue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("名前から URL を引いて削除する", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), &sqs.GetQueueUrlInput{
				QueueName: awssdk.String(orphanQueueName),
			}).Return(&sqs.GetQueueUrlOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), &sqs.DeleteQueueInput{
				QueueUrl: awssdk.String(testQueueURL),
			}).Return(&sqs.DeleteQueueOutput{}, nil)

			require.NoError(t, r.deleteQueue(t.Context(), orphanQueueName))
		})

		t.Run("受信先が既に無ければ成功として扱う", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), gomock.Any()).Return(nil, &sqstypes.QueueDoesNotExist{})

			require.NoError(t, r.deleteQueue(t.Context(), orphanQueueName))
		})

		t.Run("URL を引いた後に消えていても成功として扱う", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueUrlOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(nil, &sqstypes.QueueDoesNotExist{})

			require.NoError(t, r.deleteQueue(t.Context(), orphanQueueName))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("URL の解決に失敗したら削除を試みない", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			require.ErrorIs(t, r.deleteQueue(t.Context(), orphanQueueName), apperror.ErrUnavailable)
		})

		t.Run("削除の失敗は返す", func(t *testing.T) {
			t.Parallel()

			r, m := newReclaimer(t)
			m.sqs.EXPECT().GetQueueUrl(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueUrlOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			require.ErrorIs(t, r.deleteQueue(t.Context(), orphanQueueName), apperror.ErrUnavailable)
		})
	})
}

func Test_queueNameFromEndpoint(t *testing.T) {
	t.Parallel()

	assert.Equal(t, orphanQueueName, queueNameFromEndpoint(testQueueARN))
	assert.Empty(t, queueNameFromEndpoint("no-colon"))
	assert.Empty(t, queueNameFromEndpoint("arn:aws:sqs:us-east-1:000000000000:"))
	assert.Empty(t, queueNameFromEndpoint(""))
}

func Test_queueBelongsTo(t *testing.T) {
	t.Parallel()

	assert.True(t, queueBelongsTo(orphanQueueName, "inst-1"))
	// prefix を変えた後の旧世代の残骸にも、識別子の側で引き当てられる。
	assert.True(t, queueBelongsTo("realtime-old-inst-1", "inst-1"))
	// 前方一致では取り違える組み合わせ（識別子が別 instance の接頭辞になっている）を弾く。
	assert.False(t, queueBelongsTo("realtime-test-inst-11", "inst-1"))
	assert.False(t, queueBelongsTo("realtime-test-other", "inst-1"))
	assert.False(t, queueBelongsTo("", "inst-1"))
}

func Test_candidateQueues(t *testing.T) {
	t.Parallel()

	// 設定から導ける名前は必ず候補に入り、重複は畳まれ、順序は決まる。
	assert.Equal(t, []string{"a", "b"}, candidateQueues("b", []string{"a", "a"}))
	assert.Equal(t, []string{orphanQueueName}, candidateQueues(orphanQueueName, nil))
}

func Test_isUnsubscribable(t *testing.T) {
	t.Parallel()

	assert.True(t, isUnsubscribable(testSubARN))
	assert.False(t, isUnsubscribable("PendingConfirmation"))
	assert.False(t, isUnsubscribable(""))
}

func Test_queueGone(t *testing.T) {
	t.Parallel()

	assert.True(t, queueGone(&sqstypes.QueueDoesNotExist{}))
	assert.False(t, queueGone(errSubstrate))
	assert.False(t, queueGone(nil))
}
