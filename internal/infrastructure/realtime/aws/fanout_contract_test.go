package aws_test

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	dynamotestkit "go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/infrastructure/eventlog"
	eventlogdynamo "go-boilerplate/internal/infrastructure/eventlog/dynamodb"
	"go-boilerplate/internal/infrastructure/realtime/aws"
	"go-boilerplate/internal/infrastructure/realtime/local"
	"go-boilerplate/internal/infrastructure/realtime/testkit"
	"go-boilerplate/internal/observability"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/uuid"
)

// fanoutFixture は、実 DynamoDB Local の EventLog と実 GoAWS の topic に N 個の instance を subscribe した状態です。
type fanoutFixture struct {
	clients  aws.Clients
	topicARN string
	log      rt.EventLogStore
	subs     []rt.InstanceSubscription
}

func newFanoutFixture(t *testing.T, instances int) *fanoutFixture {
	t.Helper()

	dynamo := dynamotestkit.NewTestClient(t)
	table := dynamotestkit.TableName(t, "fanout_event_log")
	require.NoError(t, dynamodbclient.EnsureTable(t.Context(), dynamo, eventlogdynamo.TableSpec(table)))
	dynamotestkit.DeleteOnCleanup(t, dynamo, table)

	clients := testkit.NewTestClients(t)
	f := &fanoutFixture{
		clients:  clients,
		topicARN: testkit.CreateTopic(t, clients, testkit.Name(t, "fanout")),
		log:      eventlog.New(dynamo, table, observability.NewNoopTracerFactory(t)),
	}

	prefix := testkit.Name(t, "inst")
	for i := range instances {
		sub := aws.NewInstanceSubscription(
			clients.SNS, clients.SQS, aws.SubscriptionTarget{TopicARN: f.topicARN, QueuePrefix: prefix},
			local.NewQueueAttributes(), observability.NewNoopTracerFactory(t),
		)
		require.NoError(t, sub.Provision(t.Context(), rt.InstanceID("i"+strconv.Itoa(i))))
		testkit.TeardownOnCleanup(t, sub)
		f.subs = append(f.subs, sub)
	}

	return f
}

func (f *fanoutFixture) publisher(t *testing.T, topicARN string) publisherbndry.Publisher {
	t.Helper()

	return aws.NewPublisher(f.log, f.clients.SNS, topicARN,
		observability.NewNoopTracerFactory(t), observability.NewNoopRealtimeMetrics(t))
}

// message は、stream の先頭位置（sequence 1）の event を payload に持つ outbox message を返します。
func (f *fanoutFixture) message(t *testing.T, stream rt.StreamID) publisherbndry.Message {
	t.Helper()

	const seq rt.Sequence = 1

	t.Helper()

	id, err := uuid.New()
	require.NoError(t, err)

	payload, err := rt.DeliveryEvent{
		EventID: id.String(), StreamID: stream, Sequence: seq, Type: "inquiry.message.appended.v1",
		OccurredAt: time.Now().UTC(), SchemaVersion: 1, Payload: json.RawMessage(`{"seq":` + seq.String() + `}`),
	}.MarshalJSON()
	require.NoError(t, err)

	return publisherbndry.Message{MessageID: id, EventType: "inquiry.message.appended.v1", Payload: payload}
}

// drain は、sub に届いている通知を（受信の窓 1 回分だけ）集めて削除し、返します。
func drain(t *testing.T, sub rt.InstanceSubscription) []rt.Notification {
	t.Helper()

	got, err := sub.Receive(t.Context(), 10)
	require.NoError(t, err)

	for _, n := range got {
		require.NoError(t, sub.Delete(t.Context(), n))
	}

	return got
}

// TestFanoutContract は、親受入基準 1（1 publish が N 個の serve instance queue へ届く）と 13（emulator で fan-out が
// 成立する）、そして RawMessageDelivery で本文が生の wakeup と一致することを、実 EventLog + 実 emulator で固定します。
func TestFanoutContract(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1 publish が 3 instance の queue すべてへ 1 件ずつ届き、本文は eventId / streamId / sequence だけ", func(t *testing.T) {
			t.Parallel()

			f := newFanoutFixture(t, 3)
			m := f.message(t, "stream-1")
			require.NoError(t, f.publisher(t, f.topicARN).Publish(t.Context(), m))

			for i, sub := range f.subs {
				got := drain(t, sub)
				require.Len(t, got, 1, "instance %d", i)
				assert.Equal(t, rt.Notification{
					Kind: rt.KindWakeup, Wakeup: rt.Wakeup{EventID: m.MessageID.String(), StreamID: "stream-1", Sequence: 1}, Receipt: got[0].Receipt,
				}, got[0])
			}

			stored, ok, err := f.log.Find(t.Context(), "stream-1", 1)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, m.MessageID.String(), stored.EventID)
		})
	})
}

// TestPublishRetryContract は、親受入基準 3（SNS 成功後の outbox mark 失敗でも event が二重配送されない）を固定します。
// mark の失敗は relay の再 claim を生むだけなので、ここでは「同じ message を再び Publish する」ことで再現し、
// EventLog に 2 件目が入らないこと（重複 wakeup は届くが、client に配られるのは EventLog の 1 件）を確かめます。
func TestPublishRetryContract(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SNS 成功後に mark が失敗して再 claim されても、EventLog の同じ位置には 1 件しか無い", func(t *testing.T) {
			t.Parallel()

			f := newFanoutFixture(t, 1)
			p := f.publisher(t, f.topicARN)
			m := f.message(t, "stream-1")

			require.NoError(t, p.Publish(t.Context(), m))
			require.NoError(t, p.Publish(t.Context(), m), "再 claim: append は冪等に成功し、wakeup が再び publish される")

			res, err := f.log.ReadAfter(t.Context(), rt.ReadAfterQuery{StreamID: "stream-1", After: 0, Limit: 10})
			require.NoError(t, err)
			require.Len(t, res.Events, 1, "EventLog に 2 件目は入らない")
			assert.Equal(t, m.MessageID.String(), res.Events[0].EventID)

			got := drain(t, f.subs[0])
			assert.GreaterOrEqual(t, len(got), 1)
			for _, n := range got {
				assert.Equal(t, rt.Sequence(1), n.Wakeup.Sequence, "届く wakeup はすべて同じ位置を指す（読み直しに畳まれる）")
			}
		})

		t.Run("append 済みで SNS が失敗しても retryable で、再実行で wakeup だけが届く", func(t *testing.T) {
			t.Parallel()

			f := newFanoutFixture(t, 1)
			m := f.message(t, "stream-2")

			err := f.publisher(t, "arn:aws:sns:us-east-1:000000000000:does-not-exist").Publish(t.Context(), m)
			require.ErrorIs(t, err, apperror.ErrRetryable)

			_, ok, err := f.log.Find(t.Context(), "stream-2", 1)
			require.NoError(t, err)
			require.True(t, ok, "append は SNS の前に成立している")

			require.NoError(t, f.publisher(t, f.topicARN).Publish(t.Context(), m))

			res, err := f.log.ReadAfter(t.Context(), rt.ReadAfterQuery{StreamID: "stream-2", After: 0, Limit: 10})
			require.NoError(t, err)
			require.Len(t, res.Events, 1)

			got := drain(t, f.subs[0])
			require.Len(t, got, 1)
			assert.Equal(t, rt.Sequence(1), got[0].Wakeup.Sequence)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ位置に別の event を publish すると permanent で、EventLog は先勝ちのまま", func(t *testing.T) {
			t.Parallel()

			f := newFanoutFixture(t, 1)
			p := f.publisher(t, f.topicARN)
			first := f.message(t, "stream-3")
			second := f.message(t, "stream-3")

			require.NoError(t, p.Publish(t.Context(), first))

			err := p.Publish(t.Context(), second)
			require.ErrorIs(t, err, apperror.ErrPermanent)
			require.ErrorIs(t, err, rt.ErrSequenceConflict)

			stored, ok, err := f.log.Find(t.Context(), "stream-3", 1)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, first.MessageID.String(), stored.EventID)
		})
	})
}
