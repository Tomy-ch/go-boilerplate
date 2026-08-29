package aws

import (
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	mock_aws "go-boilerplate/internal/infrastructure/realtime/aws/mock"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

const (
	testTopicARN = "arn:aws:sns:us-east-1:000000000000:topic"
	testQueueURL = "http://localhost:4100/000000000000/realtime-test-inst-1"
	testQueueARN = "arn:aws:sqs:us-east-1:000000000000:realtime-test-inst-1"
	testSubARN   = testTopicARN + ":sub-1"
)

var errSubstrate = xerrors.New("substrate failed")

type subscriptionMocks struct {
	sns *mock_aws.MockSNSAPI
	sqs *mock_aws.MockSQSAPI
}

// emptyAttributes は、属性を 1 つも設定しない QueueAttributes です。
type emptyAttributes struct{}

func (emptyAttributes) Build(string) (map[string]string, error) { return map[string]string{}, nil }

// failingAttributes は、属性の組み立てに失敗する QueueAttributes です。
type failingAttributes struct{}

func (failingAttributes) Build(string) (map[string]string, error) { return nil, errSubstrate }

func newSubscription(t *testing.T) (*subscription, subscriptionMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := subscriptionMocks{sns: mock_aws.NewMockSNSAPI(ctrl), sqs: mock_aws.NewMockSQSAPI(ctrl)}
	s, ok := NewInstanceSubscription(
		m.sns, m.sqs, SubscriptionTarget{TopicARN: testTopicARN, QueuePrefix: "realtime-test"},
		NewQueueAttributes(QueueAttributesInput{TopicARN: testTopicARN}), observability.NewNoopTracerFactory(t),
	).(*subscription)
	require.True(t, ok)

	return s, m
}

// expectProvision は、Provision が成功する呼び出し列を期待します。
func expectProvision(m subscriptionMocks) {
	m.sqs.EXPECT().CreateQueue(gomock.Any(), gomock.AssignableToTypeOf(&sqs.CreateQueueInput{})).
		DoAndReturn(func(_ any, in *sqs.CreateQueueInput, _ ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error) {
			return &sqs.CreateQueueOutput{
				QueueUrl: awssdk.String("http://localhost:4100/000000000000/" + awssdk.ToString(in.QueueName)),
			}, nil
		})
	m.sqs.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).
		Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{"QueueArn": testQueueARN}}, nil)
	m.sqs.EXPECT().SetQueueAttributes(gomock.Any(), gomock.Any()).Return(&sqs.SetQueueAttributesOutput{}, nil)
	m.sns.EXPECT().
		Subscribe(gomock.Any(), gomock.Any()).
		Return(&sns.SubscribeOutput{SubscriptionArn: awssdk.String(testSubARN)}, nil)
	m.sns.EXPECT().
		SetSubscriptionAttributes(gomock.Any(), gomock.Any()).
		Return(&sns.SetSubscriptionAttributesOutput{}, nil)
}

func TestNewInstanceSubscription(t *testing.T) {
	t.Parallel()

	s, _ := newSubscription(t)
	assert.NotNil(t, s)
}

func TestQueueName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("prefix と instance の識別子を - で繋ぐ", func(t *testing.T) {
			t.Parallel()

			name, err := QueueName("realtime-local", "0123")
			require.NoError(t, err)
			assert.Equal(t, "realtime-local-0123", name)
		})

		t.Run("80 文字ちょうどは通る", func(t *testing.T) {
			t.Parallel()

			_, err := QueueName(strings.Repeat("p", 43), rt.InstanceID(strings.Repeat("i", 36)))
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("81 文字は ErrInvalidQueueName", func(t *testing.T) {
			t.Parallel()

			_, err := QueueName(strings.Repeat("p", 44), rt.InstanceID(strings.Repeat("i", 36)))
			require.ErrorIs(t, err, ErrInvalidQueueName)
		})

		t.Run("使えない文字は ErrInvalidQueueName", func(t *testing.T) {
			t.Parallel()

			_, err := QueueName("realtime.local", "0123")
			require.ErrorIs(t, err, ErrInvalidQueueName)
		})
	})
}

func Test_subscription_Provision(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("queue 作成 → ARN 解決 → 属性設定 → subscribe → RawMessageDelivery の順に呼び、状態を保持する", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			gomock.InOrder(
				m.sqs.EXPECT().CreateQueue(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, in *sqs.CreateQueueInput, _ ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error) {
						assert.Equal(t, "realtime-test-inst-1", awssdk.ToString(in.QueueName))

						return &sqs.CreateQueueOutput{QueueUrl: awssdk.String(testQueueURL)}, nil
					}),
				m.sqs.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).
					Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{"QueueArn": testQueueARN}}, nil),
				m.sqs.EXPECT().SetQueueAttributes(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, in *sqs.SetQueueAttributesInput, _ ...func(*sqs.Options)) (*sqs.SetQueueAttributesOutput, error) {
						assert.Equal(t, testQueueURL, awssdk.ToString(in.QueueUrl))
						assert.Contains(t, in.Attributes["Policy"], testTopicARN)

						return &sqs.SetQueueAttributesOutput{}, nil
					}),
				m.sns.EXPECT().Subscribe(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, in *sns.SubscribeInput, _ ...func(*sns.Options)) (*sns.SubscribeOutput, error) {
						assert.Equal(t, testTopicARN, awssdk.ToString(in.TopicArn))
						assert.Equal(t, "sqs", awssdk.ToString(in.Protocol))
						assert.Equal(t, testQueueARN, awssdk.ToString(in.Endpoint))
						assert.True(t, in.ReturnSubscriptionArn)

						return &sns.SubscribeOutput{SubscriptionArn: awssdk.String(testSubARN)}, nil
					}),
				m.sns.EXPECT().SetSubscriptionAttributes(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, in *sns.SetSubscriptionAttributesInput, _ ...func(*sns.Options)) (*sns.SetSubscriptionAttributesOutput, error) {
						assert.Equal(t, testSubARN, awssdk.ToString(in.SubscriptionArn))
						assert.Equal(t, "RawMessageDelivery", awssdk.ToString(in.AttributeName))
						assert.Equal(t, "true", awssdk.ToString(in.AttributeValue))

						return &sns.SetSubscriptionAttributesOutput{}, nil
					}),
			)

			require.NoError(t, s.Provision(t.Context(), "inst-1"))
			assert.Equal(t, testQueueURL, s.queueURL)
			assert.Equal(t, testQueueARN, s.queueARN)
			assert.Equal(t, testSubARN, s.subscriptionARN)
		})

		t.Run("同じ instance で再度呼んでも substrate を叩かない", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)

			require.NoError(t, s.Provision(t.Context(), "inst-1"))
			require.NoError(t, s.Provision(t.Context(), "inst-1"))
		})

		t.Run("属性が空なら SetQueueAttributes を呼ばない", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			s.attrs = emptyAttributes{}
			m.sqs.EXPECT().
				CreateQueue(gomock.Any(), gomock.Any()).
				Return(&sqs.CreateQueueOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{"QueueArn": testQueueARN}}, nil)
			m.sns.EXPECT().
				Subscribe(gomock.Any(), gomock.Any()).
				Return(&sns.SubscribeOutput{SubscriptionArn: awssdk.String(testSubARN)}, nil)
			m.sns.EXPECT().
				SetSubscriptionAttributes(gomock.Any(), gomock.Any()).
				Return(&sns.SetSubscriptionAttributesOutput{}, nil)

			require.NoError(t, s.Provision(t.Context(), "inst-1"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("別の instance で呼ぶと ErrConflict", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)
			require.NoError(t, s.Provision(t.Context(), "inst-1"))

			require.ErrorIs(t, s.Provision(t.Context(), "inst-2"), apperror.ErrConflict)
		})

		t.Run("queue 名が不正なら substrate を叩かずに ErrInvalidQueueName", func(t *testing.T) {
			t.Parallel()

			s, _ := newSubscription(t)
			require.ErrorIs(t, s.Provision(t.Context(), "bad.id"), ErrInvalidQueueName)
		})

		t.Run("queue 作成の失敗は ErrUnavailable で、片付ける resource は無い", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			m.sqs.EXPECT().CreateQueue(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			require.ErrorIs(t, s.Provision(t.Context(), "inst-1"), apperror.ErrUnavailable)
			assert.Empty(t, s.queueURL)
		})

		t.Run("ARN が空で返れば ErrUnavailable で、作った queue は削除する", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			m.sqs.EXPECT().
				CreateQueue(gomock.Any(), gomock.Any()).
				Return(&sqs.CreateQueueOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().
				GetQueueAttributes(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{}}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil)

			require.ErrorIs(t, s.Provision(t.Context(), "inst-1"), apperror.ErrUnavailable)
			assert.Empty(t, s.queueURL)
		})

		t.Run("subscribe の失敗は ErrUnavailable で、作った queue は削除する", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			m.sqs.EXPECT().
				CreateQueue(gomock.Any(), gomock.Any()).
				Return(&sqs.CreateQueueOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{"QueueArn": testQueueARN}}, nil)
			m.sqs.EXPECT().SetQueueAttributes(gomock.Any(), gomock.Any()).Return(&sqs.SetQueueAttributesOutput{}, nil)
			m.sns.EXPECT().Subscribe(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil)

			require.ErrorIs(t, s.Provision(t.Context(), "inst-1"), apperror.ErrUnavailable)
		})

		t.Run("戻し（rollback）にも失敗すれば、その失敗も合流して返す", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			m.sqs.EXPECT().CreateQueue(gomock.Any(), gomock.Any()).Return(&sqs.CreateQueueOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{}}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			err := s.Provision(t.Context(), "inst-1")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Contains(t, err.Error(), "delete instance queue")
			assert.Equal(t, testQueueURL, s.queueURL, "消せなかった queue は状態に残る")
		})

		t.Run("RawMessageDelivery の設定に失敗すれば unsubscribe と queue 削除まで戻す", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			m.sqs.EXPECT().
				CreateQueue(gomock.Any(), gomock.Any()).
				Return(&sqs.CreateQueueOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{"QueueArn": testQueueARN}}, nil)
			m.sqs.EXPECT().SetQueueAttributes(gomock.Any(), gomock.Any()).Return(&sqs.SetQueueAttributesOutput{}, nil)
			m.sns.EXPECT().
				Subscribe(gomock.Any(), gomock.Any()).
				Return(&sns.SubscribeOutput{SubscriptionArn: awssdk.String(testSubARN)}, nil)
			m.sns.EXPECT().SetSubscriptionAttributes(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)
			m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).Return(&sns.UnsubscribeOutput{}, nil)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil)

			require.ErrorIs(t, s.Provision(t.Context(), "inst-1"), apperror.ErrUnavailable)
			assert.Empty(t, s.subscriptionARN)
		})
	})
}

func Test_subscription_Receive(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("long polling で受け、種別ごとに復元する", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)
			require.NoError(t, s.Provision(t.Context(), "inst-1"))

			m.sqs.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ any, in *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
					assert.Equal(t, testQueueURL, awssdk.ToString(in.QueueUrl))
					assert.Equal(t, int32(5), in.MaxNumberOfMessages)
					assert.Equal(t, int32(20), in.WaitTimeSeconds)
					assert.Equal(t, []string{"All"}, in.MessageAttributeNames)

					return &sqs.ReceiveMessageOutput{Messages: []sqstypes.Message{
						message("wakeup", `{"eventId":"e1","streamId":"s1","sequence":"3"}`),
						message("", `{}`),
					}}, nil
				})

			got, err := s.Receive(t.Context(), 5)
			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, rt.KindWakeup, got[0].Kind)
			assert.Equal(t, rt.Sequence(3), got[0].Wakeup.Sequence)
			assert.Empty(t, got[1].Kind)
			assert.Equal(t, "receipt-1", got[1].Receipt)
		})

		t.Run("limit は SQS の上限 10 に丸める", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)
			require.NoError(t, s.Provision(t.Context(), "inst-1"))

			m.sqs.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ any, in *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
					assert.Equal(t, int32(10), in.MaxNumberOfMessages)

					return &sqs.ReceiveMessageOutput{}, nil
				}).
				Times(2)

			_, err := s.Receive(t.Context(), 100)
			require.NoError(t, err)
			_, err = s.Receive(t.Context(), 0)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未 provision なら ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s, _ := newSubscription(t)
			_, err := s.Receive(t.Context(), 1)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("substrate の失敗は ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)
			require.NoError(t, s.Provision(t.Context(), "inst-1"))
			m.sqs.EXPECT().ReceiveMessage(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			_, err := s.Receive(t.Context(), 1)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_subscription_Delete(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Receipt を receipt handle として削除する", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)
			require.NoError(t, s.Provision(t.Context(), "inst-1"))
			m.sqs.EXPECT().DeleteMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ any, in *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
					assert.Equal(t, testQueueURL, awssdk.ToString(in.QueueUrl))
					assert.Equal(t, "r-9", awssdk.ToString(in.ReceiptHandle))

					return &sqs.DeleteMessageOutput{}, nil
				})

			require.NoError(t, s.Delete(t.Context(), rt.Notification{Receipt: "r-9"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未 provision なら ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s, _ := newSubscription(t)
			require.ErrorIs(t, s.Delete(t.Context(), rt.Notification{Receipt: "r"}), apperror.ErrUnavailable)
		})

		t.Run("substrate の失敗は ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)
			require.NoError(t, s.Provision(t.Context(), "inst-1"))
			m.sqs.EXPECT().DeleteMessage(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			require.ErrorIs(t, s.Delete(t.Context(), rt.Notification{Receipt: "r"}), apperror.ErrUnavailable)
		})
	})
}

func Test_subscription_Teardown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("unsubscribe → queue 削除の順で片付け、状態を空に戻す", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)
			require.NoError(t, s.Provision(t.Context(), "inst-1"))
			gomock.InOrder(
				m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, in *sns.UnsubscribeInput, _ ...func(*sns.Options)) (*sns.UnsubscribeOutput, error) {
						assert.Equal(t, testSubARN, awssdk.ToString(in.SubscriptionArn))

						return &sns.UnsubscribeOutput{}, nil
					}),
				m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, in *sqs.DeleteQueueInput, _ ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error) {
						assert.Equal(t, testQueueURL, awssdk.ToString(in.QueueUrl))

						return &sqs.DeleteQueueOutput{}, nil
					}),
			)

			require.NoError(t, s.Teardown(t.Context()))
			assert.Empty(t, s.subscriptionARN)
			assert.Empty(t, s.queueURL)
			assert.Empty(t, s.instanceID)
		})

		t.Run("未 provision なら何もしない", func(t *testing.T) {
			t.Parallel()

			s, _ := newSubscription(t)
			require.NoError(t, s.Teardown(t.Context()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("unsubscribe が失敗しても queue 削除は試み、失敗をまとめて返す", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)
			require.NoError(t, s.Provision(t.Context(), "inst-1"))
			m.sns.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil)

			err := s.Teardown(t.Context())
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, testSubARN, s.subscriptionARN, "失敗した分は残し、orphan cleanup が辿れるようにする")
			assert.Empty(t, s.queueURL)
		})
	})
}

func Test_subscription_currentQueueURL(t *testing.T) {
	t.Parallel()

	s, _ := newSubscription(t)
	_, err := s.currentQueueURL()
	require.ErrorIs(t, err, apperror.ErrUnavailable)

	s.queueURL = testQueueURL
	url, err := s.currentQueueURL()
	require.NoError(t, err)
	assert.Equal(t, testQueueURL, url)
}

func Test_subscription_provision(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("queue の URL と ARN と subscription の ARN を状態に保持する", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			expectProvision(m)

			require.NoError(t, s.provision(t.Context(), "realtime-test-inst-1"))
			assert.Equal(t, "http://localhost:4100/000000000000/realtime-test-inst-1", s.queueURL)
			assert.Equal(t, testQueueARN, s.queueARN)
			assert.Equal(t, testSubARN, s.subscriptionARN)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("属性の組み立てに失敗すればその失敗を返し、queue の状態は残る", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			s.attrs = failingAttributes{}
			m.sqs.EXPECT().CreateQueue(gomock.Any(), gomock.Any()).Return(&sqs.CreateQueueOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{"QueueArn": testQueueARN}}, nil)

			require.ErrorIs(t, s.provision(t.Context(), "realtime-test-inst-1"), errSubstrate)
			assert.Equal(t, testQueueURL, s.queueURL)
		})

		t.Run("subscribe が空の ARN を返せば ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			m.sqs.EXPECT().CreateQueue(gomock.Any(), gomock.Any()).Return(&sqs.CreateQueueOutput{QueueUrl: awssdk.String(testQueueURL)}, nil)
			m.sqs.EXPECT().GetQueueAttributes(gomock.Any(), gomock.Any()).
				Return(&sqs.GetQueueAttributesOutput{Attributes: map[string]string{"QueueArn": testQueueARN}}, nil)
			m.sqs.EXPECT().SetQueueAttributes(gomock.Any(), gomock.Any()).Return(&sqs.SetQueueAttributesOutput{}, nil)
			m.sns.EXPECT().Subscribe(gomock.Any(), gomock.Any()).Return(&sns.SubscribeOutput{}, nil)

			require.ErrorIs(t, s.provision(t.Context(), "realtime-test-inst-1"), apperror.ErrUnavailable)
		})
	})
}

func Test_subscription_teardown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("何も作っていなければ substrate を叩かずに nil", func(t *testing.T) {
			t.Parallel()

			s, _ := newSubscription(t)
			require.NoError(t, s.teardown(t.Context()))
		})

		t.Run("queue だけ残っていれば削除だけ行う", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			s.queueURL, s.queueARN = testQueueURL, testQueueARN
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(&sqs.DeleteQueueOutput{}, nil)

			require.NoError(t, s.teardown(t.Context()))
			assert.Empty(t, s.queueURL)
			assert.Empty(t, s.queueARN)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("queue の削除に失敗すれば URL を残して ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s, m := newSubscription(t)
			s.queueURL = testQueueURL
			m.sqs.EXPECT().DeleteQueue(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			require.ErrorIs(t, s.teardown(t.Context()), apperror.ErrUnavailable)
			assert.Equal(t, testQueueURL, s.queueURL)
		})
	})
}
