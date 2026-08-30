package aws_test

import (
	"strings"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/infrastructure/dynamodbclient"
	dynamotestkit "go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/infrastructure/instancelease"
	instanceleasedynamo "go-boilerplate/internal/infrastructure/instancelease/dynamodb"
	"go-boilerplate/internal/infrastructure/realtime/aws"
	"go-boilerplate/internal/infrastructure/realtime/local"
	"go-boilerplate/internal/infrastructure/realtime/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/xerrors"
)

// cleanupFixture は、実 GoAWS に 1 instance ぶんの受信先を作り、実 DynamoDB Local に lease を置いた状態です。
type cleanupFixture struct {
	clients  aws.Clients
	topicARN string
	prefix   string
	leases   rt.InstanceLeaseStore
	// crashedAt は、instance が最後に生存を報告した時刻です。
	crashedAt time.Time
	// sweepAt は、掃除を行う時刻です。crashedAt から expiry + margin を十分に過ぎています。
	sweepAt time.Time
}

func newCleanupFixture(t *testing.T) *cleanupFixture {
	t.Helper()

	dynamo := dynamotestkit.NewTestClient(t)
	table := dynamotestkit.TableName(t, "cleanup_instance_lease")
	require.NoError(t, dynamodbclient.EnsureTable(t.Context(), dynamo, instanceleasedynamo.TableSpec(table)))
	dynamotestkit.DeleteOnCleanup(t, dynamo, table)

	clients := testkit.NewTestClients(t)
	crashedAt := time.Date(2026, time.August, 30, 5, 0, 0, 0, time.UTC)

	return &cleanupFixture{
		clients:   clients,
		topicARN:  testkit.CreateTopic(t, clients, testkit.Name(t, "cleanup")),
		prefix:    testkit.Name(t, "orphan"),
		leases:    instancelease.New(dynamo, table, observability.NewNoopTracerFactory(t)),
		crashedAt: crashedAt,
		// expiry + margin を過ぎた時点。実時間の経過を待たずに回収可能な状態を作る。
		sweepAt: crashedAt.Add(ucrealtime.LeaseExpiry + ucrealtime.LeaseCleanupMargin + time.Minute),
	}
}

// crash は、id の instance が受信先を作ったまま graceful shutdown を経ずに落ちた状態を作ります。
// 停止時の片付けを呼ばないので、受信先と lease の両方が残ります。
func (f *cleanupFixture) crash(t *testing.T, id rt.InstanceID) {
	t.Helper()

	sub := aws.NewInstanceSubscription(
		f.clients.SNS, f.clients.SQS, aws.SubscriptionTarget{TopicARN: f.topicARN, QueuePrefix: f.prefix},
		local.NewQueueAttributes(), observability.NewNoopTracerFactory(t),
	)
	require.NoError(t, sub.Provision(t.Context(), id))

	require.NoError(t, f.leases.Heartbeat(t.Context(), rt.InstanceLease{
		InstanceID:  id,
		HeartbeatAt: f.crashedAt,
		ExpiresAt:   f.crashedAt.Add(ucrealtime.LeaseExpiry),
	}))
}

// sweeper は、owner の名前で掃除する OrphanSweeper を返します。
func (f *cleanupFixture) sweeper(t *testing.T, owner string) ucrealtime.OrphanSweeper {
	t.Helper()

	tf := observability.NewNoopTracerFactory(t)

	return ucrealtime.NewOrphanSweeper(
		f.leases,
		aws.NewOrphanReclaimer(
			f.clients.SNS, f.clients.SQS,
			aws.SubscriptionTarget{TopicARN: f.topicARN, QueuePrefix: f.prefix}, tf,
		),
		owner,
		fixedClock(t, f.sweepAt),
		tf,
	)
}

// fixedClock は、掃除の基準時刻を固定した Clock を返します。実時間で expiry + margin を待たずに
// 回収可能な状態を作るために使います。
func fixedClock(t *testing.T, now time.Time) clock.Clock {
	t.Helper()

	clk := mock_clock.NewMockClock(gomock.NewController(t))
	clk.EXPECT().Now().Return(now).AnyTimes()

	return clk
}

// sweeperReclaimer は、fixture の topic と prefix に対する回収の手を返します。
func (f *cleanupFixture) sweeperReclaimer(t *testing.T) rt.OrphanReclaimer {
	t.Helper()

	return aws.NewOrphanReclaimer(
		f.clients.SNS, f.clients.SQS,
		aws.SubscriptionTarget{TopicARN: f.topicARN, QueuePrefix: f.prefix}, observability.NewNoopTracerFactory(t),
	)
}

// queueExists は、id の instance の受信先が実在するかを substrate へ問い合わせます。
func (f *cleanupFixture) queueExists(t *testing.T, id rt.InstanceID) bool {
	t.Helper()

	name, err := aws.QueueName(f.prefix, id)
	require.NoError(t, err)

	_, err = f.clients.SQS.GetQueueUrl(t.Context(), &sqs.GetQueueUrlInput{QueueName: awssdk.String(name)})
	if err == nil {
		return true
	}

	var gone *sqstypes.QueueDoesNotExist
	require.True(t, xerrors.As(err, &gone), "queue の存在確認が想定外の失敗をしました: %v", err)

	return false
}

// subscriptionCount は、topic に残っている subscription のうち id の受信先を指すものの数です。
func (f *cleanupFixture) subscriptionCount(t *testing.T, id rt.InstanceID) int {
	t.Helper()

	name, err := aws.QueueName(f.prefix, id)
	require.NoError(t, err)

	out, err := f.clients.SNS.ListSubscriptionsByTopic(t.Context(), &sns.ListSubscriptionsByTopicInput{
		TopicArn: awssdk.String(f.topicARN),
	})
	require.NoError(t, err)

	count := 0
	for _, s := range out.Subscriptions {
		if strings.HasSuffix(awssdk.ToString(s.Endpoint), ":"+name) {
			count++
		}
	}

	return count
}

// leaseExists は、id の lease が残っているかを返します。
func (f *cleanupFixture) leaseExists(t *testing.T, id rt.InstanceID) bool {
	t.Helper()

	// 掃除時刻より後を基準にすれば、期限切れの lease は必ず列挙に現れる。
	leases, err := f.leases.ListExpired(t.Context(), f.sweepAt.Add(time.Hour))
	require.NoError(t, err)

	for _, l := range leases {
		if l.InstanceID == id {
			return true
		}
	}

	return false
}

// TestOrphanCleanupContract は、crash した instance の受信先と lease が実 substrate 上で回収されること、
// 同時に走った掃除役のうち 1 つだけが引き受けること、繰り返し実行しても同じ状態に収束することを確かめる。
// 親 issue の受入基準「crash 後の queue / subscription を安全に回収する」に対応する。
func TestOrphanCleanupContract(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("crash した instance の subscription・queue・lease を回収する", func(t *testing.T) {
			t.Parallel()

			f := newCleanupFixture(t)
			f.crash(t, "i-crashed")
			require.True(t, f.queueExists(t, "i-crashed"))
			require.Equal(t, 1, f.subscriptionCount(t, "i-crashed"))

			got, err := f.sweeper(t, "job-a").Sweep(t.Context())
			require.NoError(t, err)
			assert.Equal(t, ucrealtime.SweepResult{Detected: 1, Claimed: 1, Reclaimed: 1}, got)

			assert.False(t, f.queueExists(t, "i-crashed"))
			assert.Equal(t, 0, f.subscriptionCount(t, "i-crashed"))
			assert.False(t, f.leaseExists(t, "i-crashed"))
		})

		t.Run("2 つの掃除役が同時に走っても 1 つしか引き受けない", func(t *testing.T) {
			t.Parallel()

			f := newCleanupFixture(t)
			f.crash(t, "i-contended")

			// 先行の掃除役が「引き受けたが、まだ回収を終えていない」状態を実 substrate 上に作る。
			// Sweep を完走させると lease まで消えてしまい、後続は列挙が空になるだけで、
			// 引き受けの条件式による相互排除を一度も通らない。
			cutoff := f.sweepAt.Add(-ucrealtime.LeaseCleanupMargin)
			claimed, err := f.leases.AcquireCleanup(t.Context(), rt.CleanupClaim{
				InstanceID:    "i-contended",
				Owner:         "job-a",
				ExpiredBefore: cutoff,
				Now:           f.sweepAt,
				OwnerUntil:    f.sweepAt.Add(ucrealtime.LeaseCleanupOwnershipTTL),
			})
			require.NoError(t, err)
			require.True(t, claimed)

			// 後続は同じ lease を見つけるが、引き受けられないので受信先に触らず見送る。
			got, err := f.sweeper(t, "job-b").Sweep(t.Context())
			require.NoError(t, err)
			assert.Equal(t, ucrealtime.SweepResult{Detected: 1, Skipped: 1}, got)
			assert.True(t, f.queueExists(t, "i-contended"))
			assert.True(t, f.leaseExists(t, "i-contended"))
		})

		t.Run("繰り返し実行しても同じ状態に収束する", func(t *testing.T) {
			t.Parallel()

			f := newCleanupFixture(t)
			f.crash(t, "i-idempotent")

			_, err := f.sweeper(t, "job-a").Sweep(t.Context())
			require.NoError(t, err)

			again, err := f.sweeper(t, "job-a").Sweep(t.Context())
			require.NoError(t, err)
			assert.Equal(t, ucrealtime.SweepResult{}, again)
		})

		t.Run("期限切れが安全余裕の内側の instance には触らない", func(t *testing.T) {
			t.Parallel()

			f := newCleanupFixture(t)
			f.crash(t, "i-recent")
			// 期限は切れているが安全余裕を過ぎていない時刻で掃除する。
			f.sweepAt = f.crashedAt.Add(ucrealtime.LeaseExpiry + time.Minute)

			got, err := f.sweeper(t, "job-a").Sweep(t.Context())
			require.NoError(t, err)
			assert.Equal(t, ucrealtime.SweepResult{}, got)

			assert.True(t, f.queueExists(t, "i-recent"))
			assert.True(t, f.leaseExists(t, "i-recent"))
		})
	})
}

// TestReceivingEndGoneContract は、外から受信先を消された instance の Receive / Delete が
// ErrReceivingEndGone を返すことを実 substrate に対して確かめる。
//
// この写像はモックでは検証できない。テストが `&sqstypes.QueueDoesNotExist{}` を注入する限り、
// emulator が別のエラー形を返しても全緑のままで、受信ループは消えた受信先に対して永久に空振りする。
// 復旧経路（作り直し）はこの分類を入口にしているので、入口が実基盤で成立することをここで締める。
func TestReceivingEndGoneContract(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受信先を外から消されたら ErrReceivingEndGone を返す", func(t *testing.T) {
			t.Parallel()

			f := newCleanupFixture(t)
			sub := aws.NewInstanceSubscription(
				f.clients.SNS, f.clients.SQS, aws.SubscriptionTarget{TopicARN: f.topicARN, QueuePrefix: f.prefix},
				local.NewQueueAttributes(), observability.NewNoopTracerFactory(t),
			)
			require.NoError(t, sub.Provision(t.Context(), "i-gone"))

			// 生きているうちは通常どおり受け取れる（空でも成功）。
			_, err := sub.Receive(t.Context(), 1)
			require.NoError(t, err)

			// orphan cleanup が誤って回収した状態を、実 substrate 上で作る。
			require.NoError(t, f.sweeperReclaimer(t).Reclaim(t.Context(), "i-gone"))

			_, err = sub.Receive(t.Context(), 1)
			require.ErrorIs(t, err, rt.ErrReceivingEndGone)

			err = sub.Delete(t.Context(), rt.Notification{Receipt: "r"})
			require.ErrorIs(t, err, rt.ErrReceivingEndGone)
		})
	})
}
