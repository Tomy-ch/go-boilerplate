package module

import (
	"context"

	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/infrastructure/realtimesecret"
	"go-boilerplate/internal/infrastructure/streamticket"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

// RealtimeAdapterModule は、feature の realtime adapter が要る最小の seam を提供する fx.Module です。
// ticket を発行するまでの経路（DynamoDB クライアント・ticket store・生値の生成・発行の usecase）だけを
// 持ち、SSE の受信側（stream handler・consumer・fan-out・起動 probe・lease heartbeat）は持ちません。
//
// 分けてあるのは、設計正本 docs/design/realtime-delivery.md の「Zero adapters, zero runtime」を
// 構造で表すためです。feature adapter を 1 つも持たない graph はこの module 自体を結線せず、
// adapter を持つ graph は runtime を伴わずにこれだけを結線できます。runtime が要る graph は
// realtimeModule() を選び、そちらがこの module を合成します——両方を同時に結線してはいけません
// （fx は module を重複排除しないため、同じ型の二重提供になります）。
//
// 採番境界（realtime.SequenceAllocator）はここにはありません。PostgreSQL 実装であり
// persistenceModule() が既に提供しています。
func RealtimeAdapterModule() fx.Option {
	return fx.Module("realtime_adapter",
		fx.Provide(
			provideRealtimeClient,
			provideStreamTicketStore,
			provideRealtimeSecretGenerator,
			provideTicketIssuer,
		),
	)
}

// provideRealtimeClient は、Realtime Delivery の store が共有する DynamoDB クライアントを組み立てます。
// 資格情報を解決できない場合はエラーを返し、app.Start を失敗させます。
func provideRealtimeClient(
	cfg *config.RealtimeConfig,
	epCfg *config.EndpointConfig,
	outbound *observability.OutboundHTTPClient,
) (*awsdynamodb.Client, error) {
	return dynamodbclient.New(context.Background(), dynamodbclient.Config{
		Endpoint:        epCfg.Realtime(),
		Region:          cfg.Region(),
		AccessKeyID:     cfg.AccessKeyID(),
		SecretAccessKey: cfg.SecretAccessKey(),
		HTTPClient:      outbound,
	})
}

func provideStreamTicketStore(
	c *awsdynamodb.Client,
	cfg *config.RealtimeConfig,
	tf observability.TracerFactory,
) rt.StreamTicketStore {
	return streamticket.New(c, cfg.StreamTicketTable(), tf)
}

func provideRealtimeSecretGenerator() rt.SecretGenerator {
	return realtimesecret.New()
}

func provideTicketIssuer(
	store rt.StreamTicketStore, secrets rt.SecretGenerator, clk clock.Clock, tf observability.TracerFactory,
) ucrealtime.TicketIssuer {
	return ucrealtime.NewTicketIssuer(store, secrets, clk, tf)
}
