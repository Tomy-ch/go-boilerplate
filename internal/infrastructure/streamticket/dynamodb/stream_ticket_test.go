package dynamodb

import (
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/realtime"
)

var (
	issuedAt = time.Date(2026, time.August, 29, 1, 0, 0, 0, time.UTC)
	expires  = issuedAt.Add(5 * time.Minute)
)

func newStore(t *testing.T) *store {
	t.Helper()

	c := testkit.NewTestClient(t)
	table := testkit.TableName(t, "stream_ticket")
	require.NoError(t, dynamodbclient.EnsureTable(t.Context(), c, TableSpec(table)))
	testkit.DeleteOnCleanup(t, c, table)

	return &store{c: c, table: table, tracer: observability.NewNoopTracerFactory(t).Infra()}
}

func ticket(hash realtime.TicketHash, subject string, destination realtime.StreamID) realtime.StreamTicket {
	return realtime.StreamTicket{
		Hash: hash, Subject: subject, Destination: destination, Scope: "read", InitialCursor: 3,
		IssuedAt: issuedAt, ExpiresAt: expires,
	}
}

// waitIndexed は、GSI（結果整合）に item が載るのを短く待ちます。
func waitIndexed(t *testing.T, s *store, subject string, destination realtime.StreamID) {
	t.Helper()

	require.Eventually(t, func() bool {
		out, err := s.c.Query(t.Context(), &dynamodb.QueryInput{
			TableName:              aws.String(s.table),
			IndexName:              aws.String(indexBySubjectDestination),
			KeyConditionExpression: aws.String(attrSubject + " = :s AND " + attrDestination + " = :d"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":s": &types.AttributeValueMemberS{Value: subject},
				":d": &types.AttributeValueMemberS{Value: string(destination)},
			},
		})

		return err == nil && len(out.Items) > 0
	}, 5*time.Second, 50*time.Millisecond)
}

func TestNew(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, New(testkit.NewTestClient(t), "realtime_stream_ticket_test", observability.NewNoopTracerFactory(t)))
}

func Test_store_Save(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存して hash で読み戻せる", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Save(t.Context(), ticket("h1", "alice", "stream-a")))

			got, ok, err := s.Find(t.Context(), "h1", issuedAt)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, ticket("h1", "alice", "stream-a"), got)
		})

		t.Run("同じ hash への再保存は上書きになる", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Save(t.Context(), ticket("h1", "alice", "stream-a")))
			require.NoError(t, s.Save(t.Context(), ticket("h1", "alice", "stream-b")))

			got, ok, err := s.Find(t.Context(), "h1", issuedAt)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, realtime.StreamID("stream-b"), got.Destination)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("hash が空なら ErrInvalidArgument", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.ErrorIs(t, s.Save(t.Context(), ticket("", "alice", "stream-a")), apperror.ErrInvalidArgument)
		})

		t.Run("table が無ければ ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s := &store{c: testkit.NewTestClient(t), table: "test_missing_table", tracer: observability.NewNoopTracerFactory(t).Infra()}
			require.ErrorIs(t, s.Save(t.Context(), ticket("h1", "alice", "stream-a")), apperror.ErrUnavailable)
		})
	})
}

func Test_store_Find(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限内なら何度でも読める（reuse）", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Save(t.Context(), ticket("h1", "alice", "stream-a")))

			for range 3 {
				_, ok, err := s.Find(t.Context(), "h1", expires.Add(-time.Second))
				require.NoError(t, err)
				assert.True(t, ok)
			}
		})

		t.Run("期限ちょうどで ok=false になる", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Save(t.Context(), ticket("h1", "alice", "stream-a")))

			_, ok, err := s.Find(t.Context(), "h1", expires)
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("無い hash は ok=false", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)

			_, ok, err := s.Find(t.Context(), "nope", issuedAt)
			require.NoError(t, err)
			assert.False(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s := &store{c: testkit.NewTestClient(t), table: "test_missing_table", tracer: observability.NewNoopTracerFactory(t).Infra()}
			_, _, err := s.Find(t.Context(), "h1", issuedAt)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_store_Invalidate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("subject × destination の ticket だけを全て消す", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Save(t.Context(), ticket("h1", "alice", "stream-a")))
			require.NoError(t, s.Save(t.Context(), ticket("h2", "alice", "stream-a")))
			require.NoError(t, s.Save(t.Context(), ticket("h3", "alice", "stream-b")))
			require.NoError(t, s.Save(t.Context(), ticket("h4", "bob", "stream-a")))
			waitIndexed(t, s, "alice", "stream-a")

			require.NoError(t, s.Invalidate(t.Context(), "alice", "stream-a"))

			for hash, want := range map[realtime.TicketHash]bool{"h1": false, "h2": false, "h3": true, "h4": true} {
				_, ok, err := s.Find(t.Context(), hash, issuedAt)
				require.NoError(t, err)
				assert.Equal(t, want, ok, string(hash))
			}
		})

		t.Run("該当が無くてもエラーにならない", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Invalidate(t.Context(), "nobody", "nowhere"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			s := &store{c: testkit.NewTestClient(t), table: "test_missing_table", tracer: observability.NewNoopTracerFactory(t).Infra()}
			require.ErrorIs(t, s.Invalidate(t.Context(), "alice", "stream-a"), apperror.ErrUnavailable)
		})
	})
}

func Test_key(t *testing.T) {
	t.Parallel()

	assert.Equal(t, &types.AttributeValueMemberS{Value: "h"}, key("h")[attrTicketHash])
}

func Test_toItem(t *testing.T) {
	t.Parallel()

	item := toItem(ticket("h1", "alice", "stream-a"))
	assert.Equal(t, &types.AttributeValueMemberS{Value: "h1"}, item[attrTicketHash])
	assert.Equal(t, &types.AttributeValueMemberN{Value: "3"}, item[attrInitialCursor])
	assert.Equal(t, &types.AttributeValueMemberN{Value: strconv.FormatInt(expires.Unix(), 10)}, item[attrExpiresAt])
	assert.Equal(t, &types.AttributeValueMemberS{Value: "alice"}, item[attrSubject])
	assert.Equal(t, &types.AttributeValueMemberS{Value: "stream-a"}, item[attrDestination])
}

func Test_fromItem(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("toItem の結果を ticket に戻せる（期限は秒精度）", func(t *testing.T) {
			t.Parallel()

			got, err := fromItem(toItem(ticket("h1", "alice", "stream-a")))
			require.NoError(t, err)
			assert.Equal(t, ticket("h1", "alice", "stream-a"), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("initial_cursor が数値でなければ ErrInternal", func(t *testing.T) {
			t.Parallel()

			item := toItem(ticket("h1", "alice", "stream-a"))
			item[attrInitialCursor] = &types.AttributeValueMemberS{Value: "x"}
			_, err := fromItem(item)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("issued_at が時刻でなければ ErrInternal", func(t *testing.T) {
			t.Parallel()

			item := toItem(ticket("h1", "alice", "stream-a"))
			item[attrIssuedAt] = &types.AttributeValueMemberS{Value: "yesterday"}
			_, err := fromItem(item)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func TestTableSpec(t *testing.T) {
	t.Parallel()

	spec := TableSpec("realtime_stream_ticket_test")
	assert.Equal(t, attrExpiresAt, spec.TTLAttribute)
	assert.Equal(t, attrTicketHash, aws.ToString(spec.KeySchema[0].AttributeName))
	require.Len(t, spec.GlobalSecondaryIndexes, 1)
	assert.Equal(t, indexBySubjectDestination, aws.ToString(spec.GlobalSecondaryIndexes[0].IndexName))
	assert.Equal(t, types.ProjectionTypeKeysOnly, spec.GlobalSecondaryIndexes[0].Projection.ProjectionType)
	assert.Equal(t, attrSubject, aws.ToString(spec.GlobalSecondaryIndexes[0].KeySchema[0].AttributeName))
	assert.Equal(t, attrDestination, aws.ToString(spec.GlobalSecondaryIndexes[0].KeySchema[1].AttributeName))
}
