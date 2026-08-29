package module

import (
	"strings"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	dynamotestkit "go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/infrastructure/instancelease"
	instanceleasedynamo "go-boilerplate/internal/infrastructure/instancelease/dynamodb"
	realtimeinfra "go-boilerplate/internal/infrastructure/realtime"
	"go-boilerplate/internal/infrastructure/realtime/testkit"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// TestRealtimeProvisionerContract は、serve lifecycle の realtime 参加者（lease + instance の受信先）を実 DynamoDB Local と
// 実 GoAWS に対して往復させ、Provision で lease と queue / subscription が現れ、Teardown で消えることを固定します。
// 起動・停止の順序そのものは internal/di/server/hook の unit test が固定し、ここは substrate 上の結果を見ます。
func TestRealtimeProvisionerContract(t *testing.T) {
	t.Parallel()

	dynamo := dynamotestkit.NewTestClient(t)
	table := dynamotestkit.TableName(t, "lifecycle_lease")
	require.NoError(t, dynamodbclient.EnsureTable(t.Context(), dynamo, instanceleasedynamo.TableSpec(table)))
	dynamotestkit.DeleteOnCleanup(t, dynamo, table)

	tf := observability.NewNoopTracerFactory(t)
	leases := instancelease.New(dynamo, table, tf)
	keeper := provideLeaseKeeper(leases, system.NewClock(), tf)

	clients := testkit.NewTestClients(t)
	topicARN := testkit.CreateTopic(t, clients, testkit.Name(t, "lifecycle"))
	cfg := config.NewRealtimeConfig(config.MockConfigForTest(t))
	sub := provideInstanceSubscription(realtimeFanout{clients: clients, topicARN: topicARN}, cfg, realtimeinfra.NewEmulatorQueueAttributes(), tf)

	id, err := provideInstanceID()
	require.NoError(t, err)

	// queue 名は <prefix>-<instance id> なので、この instance の queue だけを prefix で引ける。
	prefix := cfg.QueuePrefix() + "-" + string(id)

	p := provideRealtimeProvisioner(sub, keeper, id)
	require.NoError(t, p.Provision(t.Context()))
	t.Cleanup(func() { _ = p.Teardown(t.Context()) })

	assert.True(t, leaseExists(t, leases, id), "lease が書かれている")
	assert.Len(t, queueURLs(t, clients, prefix), 1, "instance queue が 1 つある")

	require.NoError(t, p.Teardown(t.Context()))

	assert.False(t, leaseExists(t, leases, id), "lease が消えている")
	assert.Empty(t, queueURLs(t, clients, prefix), "instance queue が消えている")
}

// leaseExists は、id の lease が store にあるかを、遠い将来を基準にした期限切れ一覧で確かめます。
func leaseExists(t *testing.T, leases rt.InstanceLeaseStore, id rt.InstanceID) bool {
	t.Helper()

	got, err := leases.ListExpired(t.Context(), time.Now().Add(24*time.Hour))
	require.NoError(t, err)

	for _, l := range got {
		if l.InstanceID == id {
			return true
		}
	}

	return false
}

// queueURLs は、prefix で始まる queue の URL を返します。
func queueURLs(t *testing.T, clients realtimeinfra.Clients, prefix string) []string {
	t.Helper()

	out, err := clients.SQS.ListQueues(t.Context(), &sqs.ListQueuesInput{QueueNamePrefix: awssdk.String(prefix)})
	require.NoError(t, err)

	urls := make([]string, 0, len(out.QueueUrls))
	for _, u := range out.QueueUrls {
		if strings.Contains(u, prefix) {
			urls = append(urls, u)
		}
	}

	return urls
}
