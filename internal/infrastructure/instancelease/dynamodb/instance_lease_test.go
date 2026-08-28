package dynamodb

import (
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

var (
	now     = time.Date(2026, time.August, 29, 1, 0, 0, 0, time.UTC)
	expiry  = now.Add(2 * time.Minute)
	margin  = 5 * time.Minute
	leaseOK = realtime.InstanceLease{InstanceID: "i-1", HeartbeatAt: now, ExpiresAt: expiry}
)

func newStore(t *testing.T) *store {
	t.Helper()

	c := testkit.NewTestClient(t)
	table := testkit.TableName(t, "instance_lease")
	require.NoError(t, dynamodbclient.EnsureTable(t.Context(), c, TableSpec(table)))
	testkit.DeleteOnCleanup(t, c, table)

	return &store{c: c, table: table, tracer: observability.NewNoopTracerFactory(t).Infra()}
}

func missingStore(t *testing.T) *store {
	t.Helper()

	return &store{c: testkit.NewTestClient(t), table: "test_missing_table", tracer: observability.NewNoopTracerFactory(t).Infra()}
}

// claimAt は、asOf 時点で id の回収を owner が引き受ける要求です（expiry + margin を過ぎたものだけ）。
func claimAt(id realtime.InstanceID, owner string, asOf time.Time) realtime.CleanupClaim {
	return realtime.CleanupClaim{
		InstanceID: id, Owner: owner, ExpiredBefore: asOf.Add(-margin), Now: asOf, OwnerUntil: asOf.Add(10 * time.Minute),
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, New(testkit.NewTestClient(t), "realtime_instance_lease_test", observability.NewNoopTracerFactory(t)))
}

func Test_store_Heartbeat(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無ければ作り、あれば期限を伸ばす", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))

			later := realtime.InstanceLease{InstanceID: "i-1", HeartbeatAt: now.Add(30 * time.Second), ExpiresAt: expiry.Add(30 * time.Second)}
			require.NoError(t, s.Heartbeat(t.Context(), later))

			expired, err := s.ListExpired(t.Context(), expiry.Add(10*time.Second))
			require.NoError(t, err)
			assert.Empty(t, expired, "伸びた期限より前では期限切れにならない")

			expired, err = s.ListExpired(t.Context(), expiry.Add(time.Minute))
			require.NoError(t, err)
			require.Len(t, expired, 1)
			assert.Equal(t, later, expired[0])
		})

		t.Run("回収の引き受けは heartbeat で消えない", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))
			asOf := expiry.Add(margin + time.Second)
			ok, err := s.AcquireCleanup(t.Context(), claimAt("i-1", "job-a", asOf))
			require.NoError(t, err)
			require.True(t, ok)

			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))

			expired, err := s.ListExpired(t.Context(), asOf)
			require.NoError(t, err)
			require.Len(t, expired, 1)
			assert.Equal(t, "job-a", expired[0].CleanupOwner)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("instance id が空なら ErrInvalidArgument", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, newStore(t).Heartbeat(t.Context(), realtime.InstanceLease{}), apperror.ErrInvalidArgument)
		})

		t.Run("table が無ければ ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, missingStore(t).Heartbeat(t.Context(), leaseOK), apperror.ErrUnavailable)
		})
	})
}

func Test_store_Delete(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("削除後は期限切れの一覧に現れず、無い lease の削除も成功する", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))
			require.NoError(t, s.Delete(t.Context(), "i-1"))
			require.NoError(t, s.Delete(t.Context(), "i-1"))

			expired, err := s.ListExpired(t.Context(), expiry.Add(time.Hour))
			require.NoError(t, err)
			assert.Empty(t, expired)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, missingStore(t).Delete(t.Context(), "i-1"), apperror.ErrUnavailable)
		})
	})
}

func Test_store_ListExpired(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("asOf より前に期限を迎えた lease だけを返す", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))
			alive := realtime.InstanceLease{InstanceID: "i-2", HeartbeatAt: now, ExpiresAt: expiry.Add(time.Hour)}
			require.NoError(t, s.Heartbeat(t.Context(), alive))

			expired, err := s.ListExpired(t.Context(), expiry.Add(time.Second))
			require.NoError(t, err)
			require.Len(t, expired, 1)
			assert.Equal(t, leaseOK, expired[0])
		})

		t.Run("期限ちょうどは期限切れではない", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))

			expired, err := s.ListExpired(t.Context(), expiry)
			require.NoError(t, err)
			assert.Empty(t, expired)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			_, err := missingStore(t).ListExpired(t.Context(), now)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_store_AcquireCleanup(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限 + margin を過ぎた未回収の lease は引き受けられ、2 者目は false", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))
			asOf := expiry.Add(margin + time.Second)

			first, err := s.AcquireCleanup(t.Context(), claimAt("i-1", "job-a", asOf))
			require.NoError(t, err)
			assert.True(t, first)

			second, err := s.AcquireCleanup(t.Context(), claimAt("i-1", "job-b", asOf))
			require.NoError(t, err)
			assert.False(t, second)
		})

		t.Run("引き受けが失効していれば別の主体が引き受け直せる", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))
			asOf := expiry.Add(margin + time.Second)
			ok, err := s.AcquireCleanup(t.Context(), claimAt("i-1", "job-a", asOf))
			require.NoError(t, err)
			require.True(t, ok)

			ok, err = s.AcquireCleanup(t.Context(), claimAt("i-1", "job-b", asOf.Add(11*time.Minute)))
			require.NoError(t, err)
			assert.True(t, ok)

			expired, err := s.ListExpired(t.Context(), asOf)
			require.NoError(t, err)
			require.Len(t, expired, 1)
			assert.Equal(t, "job-b", expired[0].CleanupOwner)
		})

		t.Run("margin の内側ではまだ引き受けられない", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)
			require.NoError(t, s.Heartbeat(t.Context(), leaseOK))

			ok, err := s.AcquireCleanup(t.Context(), claimAt("i-1", "job-a", expiry.Add(margin-time.Second)))
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("lease が無ければ作らずに false", func(t *testing.T) {
			t.Parallel()

			s := newStore(t)

			ok, err := s.AcquireCleanup(t.Context(), claimAt("ghost", "job-a", now))
			require.NoError(t, err)
			assert.False(t, ok)

			expired, err := s.ListExpired(t.Context(), now.Add(time.Hour))
			require.NoError(t, err)
			assert.Empty(t, expired)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("table が無ければ ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			_, err := missingStore(t).AcquireCleanup(t.Context(), claimAt("i-1", "job-a", now))
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_key(t *testing.T) {
	t.Parallel()

	assert.Equal(t, &types.AttributeValueMemberS{Value: "i-1"}, key("i-1")[attrInstanceID])
}

func Test_nano(t *testing.T) {
	t.Parallel()

	assert.Equal(t, &types.AttributeValueMemberN{Value: "0"}, nano(time.Time{}))
	assert.Equal(t, &types.AttributeValueMemberN{Value: strconv.FormatInt(now.UnixNano(), 10)}, nano(now))
}

func Test_fromNano(t *testing.T) {
	t.Parallel()

	assert.True(t, fromNano(0).IsZero())
	assert.Equal(t, now, fromNano(now.UnixNano()))
}

func Test_fromItem(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引き受けの無い item は CleanupOwner が空", func(t *testing.T) {
			t.Parallel()

			item := map[string]types.AttributeValue{
				attrInstanceID: &types.AttributeValueMemberS{Value: "i-1"}, attrHeartbeatAt: nano(now), attrExpiresAt: nano(expiry),
			}
			got, err := fromItem(item)
			require.NoError(t, err)
			assert.Equal(t, leaseOK, got)
		})

		t.Run("引き受けのある item は owner と期限を戻す", func(t *testing.T) {
			t.Parallel()

			item := map[string]types.AttributeValue{
				attrInstanceID: &types.AttributeValueMemberS{Value: "i-1"}, attrHeartbeatAt: nano(now), attrExpiresAt: nano(expiry),
				attrCleanupOwner: &types.AttributeValueMemberS{Value: "job-a"}, attrCleanupOwnerUntil: nano(expiry.Add(time.Hour)),
			}
			got, err := fromItem(item)
			require.NoError(t, err)
			assert.Equal(t, "job-a", got.CleanupOwner)
			assert.Equal(t, expiry.Add(time.Hour), got.CleanupOwnerUntil)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("expires_at が数値でなければ ErrInternal", func(t *testing.T) {
			t.Parallel()

			item := map[string]types.AttributeValue{
				attrInstanceID: &types.AttributeValueMemberS{Value: "i-1"}, attrHeartbeatAt: nano(now),
				attrExpiresAt: &types.AttributeValueMemberS{Value: "later"},
			}
			_, err := fromItem(item)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("cleanup_owner_until が数値でなければ ErrInternal", func(t *testing.T) {
			t.Parallel()

			item := map[string]types.AttributeValue{
				attrInstanceID: &types.AttributeValueMemberS{Value: "i-1"}, attrHeartbeatAt: nano(now), attrExpiresAt: nano(expiry),
				attrCleanupOwnerUntil: &types.AttributeValueMemberS{Value: "later"},
			}
			_, err := fromItem(item)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func TestTableSpec(t *testing.T) {
	t.Parallel()

	spec := TableSpec("realtime_instance_lease_test")
	assert.Empty(t, spec.TTLAttribute, "lease は TTL で消えてはならない")
	assert.Equal(t, attrInstanceID, aws.ToString(spec.KeySchema[0].AttributeName))
}
