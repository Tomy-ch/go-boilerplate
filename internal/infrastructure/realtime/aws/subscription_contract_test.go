package aws_test

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/infrastructure/realtime/aws"
	"go-boilerplate/internal/infrastructure/realtime/local"
	"go-boilerplate/internal/infrastructure/realtime/testkit"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// TestInstanceSubscriptionContract は、実 emulator（GoAWS。環境変数で本番 SNS / SQS へ向け直せる）に対して
// Provision → 通知の受信 → Delete → Teardown の往復を確かめます。queue 属性は emulator が受け付ける集合です。
func TestInstanceSubscriptionContract(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("provision した queue に topic からの wakeup と revocation が届き、削除と片付けまで通る", func(t *testing.T) {
			t.Parallel()

			c := testkit.NewTestClients(t)
			topicARN := testkit.CreateTopic(t, c, testkit.Name(t, "topic"))
			sub := aws.NewInstanceSubscription(
				c.SNS,
				c.SQS,
				topicARN,
				testkit.Name(t, "q"),
				local.NewQueueAttributes(),
				observability.NewNoopTracerFactory(t),
			)

			require.NoError(t, sub.Provision(t.Context(), "inst1"))
			t.Cleanup(func() { _ = sub.Teardown(t.Context()) })

			notifier := aws.NewRevocationNotifier(c.SNS, topicARN, observability.NewNoopTracerFactory(t))
			require.NoError(t, notifier.NotifyRevoked(t.Context(), "subject-1", "stream-1"))

			_, err := c.SNS.Publish(t.Context(), &sns.PublishInput{
				TopicArn: awssdk.String(topicARN),
				Message:  awssdk.String(`{"eventId":"e1","streamId":"stream-1","sequence":"5"}`),
				MessageAttributes: map[string]snstypes.MessageAttributeValue{
					aws.AttrKind: {DataType: awssdk.String("String"), StringValue: awssdk.String("wakeup")},
				},
			})
			require.NoError(t, err)

			var got []rt.Notification
			for range 3 {
				batch, err := sub.Receive(t.Context(), 10)
				require.NoError(t, err)

				got = append(got, batch...)
				if len(got) >= 2 {
					break
				}
			}

			require.Len(t, got, 2)
			kinds := map[rt.NotificationKind]rt.Notification{}
			for _, n := range got {
				kinds[n.Kind] = n
				require.NoError(t, sub.Delete(t.Context(), n))
			}

			assert.Equal(t, rt.Wakeup{EventID: "e1", StreamID: "stream-1", Sequence: 5}, kinds[rt.KindWakeup].Wakeup)
			assert.Equal(
				t,
				rt.Revocation{Subject: "subject-1", Destination: "stream-1"},
				kinds[rt.KindRevocation].Revocation,
			)

			require.NoError(t, sub.Teardown(t.Context()))
			require.NoError(t, sub.Teardown(t.Context()), "2 度目は何もしない")
		})
	})
}
