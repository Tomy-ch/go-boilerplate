package module

import (
	"testing"

	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

func TestRealtimeAdapterModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 出力型をすべて要求する。fx.ValidateApp は要求された型しか解決しないため、
	// 要求が無いと結線の欠落を素通りさせる。
	var (
		client  *awsdynamodb.Client
		tickets rt.StreamTicketStore
		secrets rt.SecretGenerator
		issuer  ucrealtime.TicketIssuer
	)
	validateGraph(t, append(commonDeps(),
		InfrastructureModule(),
		RealtimeAdapterModule(),
		fx.Populate(&client, &tickets, &secrets, &issuer),
	)...)
}

func TestRealtimeAdapterModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受信側のruntimeを伴わずにticket発行までを組める", func(t *testing.T) {
			t.Parallel()

			// 受信側（stream handler・consumer・fan-out・起動 probe）を 1 つも結線せずに
			// TicketIssuer まで解決できることを見る。constructor は実行しない——実 DB や
			// DynamoDB への到達性はこの module の関心ではない。
			var issuer ucrealtime.TicketIssuer
			err := fx.ValidateApp(append(commonDeps(),
				InfrastructureModule(),
				RealtimeAdapterModule(),
				fx.Populate(&issuer),
				fx.NopLogger,
			)...)

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("runtimeのmoduleと同じgraphへ載せると二重提供で失敗する", func(t *testing.T) {
			t.Parallel()

			// realtimeModule() はこの module を合成するため、両方を結線すると同じ型が 2 回提供される。
			// 片方だけを選ぶという制約を、文書ではなく graph で固定する。
			err := fx.ValidateApp(append(commonDeps(),
				InfrastructureModule(),
				RealtimeAdapterModule(),
				realtimeModule(),
				fx.NopLogger,
			)...)

			require.Error(t, err)
		})
	})
}
