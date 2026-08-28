package dynamodb

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/realtime"
)

// newStore は、実行ごとに一意な table を DynamoDB Local に作り、それを指す store を返します。
func newStore(t *testing.T) *store {
	t.Helper()

	c := testkit.NewTestClient(t)
	table := testkit.TableName(t, "event_log")
	require.NoError(t, dynamodbclient.EnsureTable(t.Context(), c, TableSpec(table)))
	testkit.DeleteOnCleanup(t, c, table)

	return &store{c: c, table: table, tracer: observability.NewNoopTracerFactory(t).Infra()}
}

func event(stream realtime.StreamID, seq realtime.Sequence, id string) realtime.DeliveryEvent {
	return realtime.DeliveryEvent{
		EventID:       id,
		StreamID:      stream,
		Sequence:      seq,
		Type:          "inquiry.message.appended.v1",
		OccurredAt:    time.Date(2026, time.August, 29, 1, 2, 3, 456000000, time.UTC),
		SchemaVersion: 1,
		Payload:       json.RawMessage(`{"seq":` + seq.String() + `}`),
	}
}

func seed(t *testing.T, s *store, stream realtime.StreamID, n int) {
	t.Helper()

	for i := 1; i <= n; i++ {
		require.NoError(t, s.Append(t.Context(), event(stream, realtime.Sequence(i), "evt-"+strconv.Itoa(i))))
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	s := New(testkit.NewTestClient(t), "realtime_event_log_test", observability.NewNoopTracerFactory(t))
	assert.NotNil(t, s)
}

func Test_store_Append(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空いている位置へ書ける", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Append(t.Context(), event("s", 1, "evt-1")))

			got, ok, err := s.Find(t.Context(), "s", 1)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, event("s", 1, "evt-1"), got)
		})

		t.Run("同じ EventID の再 append は成功する（outbox retry に対して冪等）", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Append(t.Context(), event("s", 1, "evt-1")))
			require.NoError(t, s.Append(t.Context(), event("s", 1, "evt-1")))

			res, err := s.ReadAfter(t.Context(), realtime.ReadAfterQuery{StreamID: "s"})
			require.NoError(t, err)
			assert.Len(t, res.Events, 1)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ位置に別の EventID があれば ErrSequenceConflict を返す", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Append(t.Context(), event("s", 1, "evt-1")))

			err := s.Append(t.Context(), event("s", 1, "evt-other"))
			require.ErrorIs(t, err, realtime.ErrSequenceConflict)
		})

		t.Run("不正な封筒は書かずに拒否する", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			e := event("s", 1, "")
			require.ErrorIs(t, s.Append(t.Context(), e), realtime.ErrInvalidEvent)

			_, ok, err := s.Find(t.Context(), "s", 1)
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("table が無ければ ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			s := &store{c: testkit.NewTestClient(t), table: "test_missing_table", tracer: observability.NewNoopTracerFactory(t).Infra()}
			require.ErrorIs(t, s.Append(t.Context(), event("s", 1, "evt-1")), apperror.ErrUnavailable)
		})
	})
}

func Test_store_ReadAfter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursor より後ろを昇順に返す", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			seed(t, s, "s", 5)

			res, err := s.ReadAfter(t.Context(), realtime.ReadAfterQuery{StreamID: "s", After: 2})
			require.NoError(t, err)
			require.Len(t, res.Events, 3)
			assert.Equal(t, realtime.Sequence(3), res.Events[0].Sequence)
			assert.Equal(t, realtime.Sequence(5), res.Events[2].Sequence)
			assert.False(t, res.HasMore)
		})

		t.Run("Limit で打ち切られると HasMore が true になり、最後の sequence から続きが読める", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			seed(t, s, "s", 5)

			first, err := s.ReadAfter(t.Context(), realtime.ReadAfterQuery{StreamID: "s", Limit: 2})
			require.NoError(t, err)
			require.Len(t, first.Events, 2)
			assert.True(t, first.HasMore)

			rest, err := s.ReadAfter(t.Context(), realtime.ReadAfterQuery{StreamID: "s", After: first.Events[1].Sequence, Limit: 10})
			require.NoError(t, err)
			require.Len(t, rest.Events, 3)
			assert.Equal(t, realtime.Sequence(3), rest.Events[0].Sequence)
			assert.False(t, rest.HasMore)
		})

		t.Run("After が 0 なら先頭から読む", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			seed(t, s, "s", 2)

			res, err := s.ReadAfter(t.Context(), realtime.ReadAfterQuery{StreamID: "s"})
			require.NoError(t, err)
			require.Len(t, res.Events, 2)
			assert.Equal(t, realtime.Sequence(1), res.Events[0].Sequence)
		})

		t.Run("別 stream の event は混ざらない", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			seed(t, s, "a", 2)
			seed(t, s, "b", 3)

			res, err := s.ReadAfter(t.Context(), realtime.ReadAfterQuery{StreamID: "a"})
			require.NoError(t, err)
			assert.Len(t, res.Events, 2)
		})

		t.Run("保持期間より古い event も OccurredAt を保ったまま返す（失効の判定は usecase）", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			old := event("s", 1, "evt-old")
			old.OccurredAt = time.Now().UTC().Add(-2 * realtime.EventLogRetention).Truncate(time.Millisecond)
			require.NoError(t, s.Append(t.Context(), old))

			res, err := s.ReadAfter(t.Context(), realtime.ReadAfterQuery{StreamID: "s"})
			require.NoError(t, err)
			require.Len(t, res.Events, 1)
			assert.True(t, res.Events[0].OccurredAt.Equal(old.OccurredAt))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			s := &store{c: testkit.NewTestClient(t), table: "test_missing_table", tracer: observability.NewNoopTracerFactory(t).Infra()}
			_, err := s.ReadAfter(t.Context(), realtime.ReadAfterQuery{StreamID: "s"})
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_store_Latest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最後の event を返す", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			seed(t, s, "s", 3)

			got, ok, err := s.Latest(t.Context(), "s")
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, realtime.Sequence(3), got.Sequence)
		})

		t.Run("event が無ければ ok=false", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)

			_, ok, err := s.Latest(t.Context(), "empty")
			require.NoError(t, err)
			assert.False(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			s := &store{c: testkit.NewTestClient(t), table: "test_missing_table", tracer: observability.NewNoopTracerFactory(t).Infra()}
			_, _, err := s.Latest(t.Context(), "s")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_store_Find(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定位置の event を返す", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			seed(t, s, "s", 3)

			got, ok, err := s.Find(t.Context(), "s", 2)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, "evt-2", got.EventID)
		})

		t.Run("無い位置は ok=false", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)

			_, ok, err := s.Find(t.Context(), "s", 9)
			require.NoError(t, err)
			assert.False(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			s := &store{c: testkit.NewTestClient(t), table: "test_missing_table", tracer: observability.NewNoopTracerFactory(t).Infra()}
			_, _, err := s.Find(t.Context(), "s", 1)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_key(t *testing.T) {
	t.Parallel()

	k := key("s", 42)
	assert.Equal(t, &types.AttributeValueMemberS{Value: "s"}, k[attrStreamID])
	assert.Equal(t, &types.AttributeValueMemberN{Value: "42"}, k[attrSequence])
}

func Test_toItem(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全属性を写し、expires_at は OccurredAt + 保持期間の epoch 秒になる", func(t *testing.T) {
			t.Parallel()

			e := event("s", 7, "evt-7")
			item := toItem(e)

			assert.Equal(t, &types.AttributeValueMemberS{Value: "evt-7"}, item[attrEventID])
			assert.Equal(t, &types.AttributeValueMemberS{Value: "2026-08-29T01:02:03.456Z"}, item[attrOccurredAt])
			assert.Equal(t, &types.AttributeValueMemberN{Value: "1"}, item[attrSchemaVersion])
			assert.Equal(t, &types.AttributeValueMemberB{Value: []byte(`{"seq":7}`)}, item[attrPayload])
			wantExpires := strconv.FormatInt(e.OccurredAt.Add(realtime.EventLogRetention).Unix(), 10)
			assert.Equal(t, &types.AttributeValueMemberN{Value: wantExpires}, item[attrExpiresAt])
		})

		t.Run("payload が空なら属性を持たない", func(t *testing.T) {
			t.Parallel()

			e := event("s", 1, "evt-1")
			e.Payload = nil
			_, has := toItem(e)[attrPayload]
			assert.False(t, has)
		})
	})
}

func Test_fromItem(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("toItem の結果を封筒に戻せる", func(t *testing.T) {
			t.Parallel()

			e := event("s", 7, "evt-7")
			got, err := fromItem(toItem(e))
			require.NoError(t, err)
			assert.Equal(t, e, got)
		})

		t.Run("payload 属性が無ければ nil になる", func(t *testing.T) {
			t.Parallel()

			e := event("s", 1, "evt-1")
			e.Payload = nil
			got, err := fromItem(toItem(e))
			require.NoError(t, err)
			assert.Nil(t, got.Payload)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("sequence が数値でなければ ErrInternal を返す", func(t *testing.T) {
			t.Parallel()

			item := toItem(event("s", 1, "evt-1"))
			item[attrSequence] = &types.AttributeValueMemberS{Value: "x"}
			_, err := fromItem(item)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("occurred_at が時刻でなければ ErrInternal を返す", func(t *testing.T) {
			t.Parallel()

			item := toItem(event("s", 1, "evt-1"))
			item[attrOccurredAt] = &types.AttributeValueMemberS{Value: "yesterday"}
			_, err := fromItem(item)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func TestTableSpec(t *testing.T) {
	t.Parallel()

	spec := TableSpec("realtime_event_log_test")
	assert.Equal(t, "realtime_event_log_test", spec.Name)
	assert.Equal(t, attrExpiresAt, spec.TTLAttribute)
	assert.Equal(t, types.KeyTypeHash, spec.KeySchema[0].KeyType)
	assert.Equal(t, attrStreamID, aws.ToString(spec.KeySchema[0].AttributeName))
	assert.Equal(t, types.KeyTypeRange, spec.KeySchema[1].KeyType)
	assert.Equal(t, attrSequence, aws.ToString(spec.KeySchema[1].AttributeName))
}
