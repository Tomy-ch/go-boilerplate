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

// RealtimeAdapterModule は、feature の realtime adapter が要る最小の seam——ticket を発行するまでの
// 経路——を提供する fx.Module です。SSE の受信側は持ちません。realtimeModule() がこの module を
// 合成するので、両方を同じ graph に結線してはいけません。
// 内訳・分割の理由・採番境界の所在は internal/di/module/README.md の Module List を参照。
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
