package module

import (
	"context"

	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	oapiauth "go-boilerplate/internal/controller/httpstack/oapi/auth"
	"go-boilerplate/internal/controller/stream"
	streamauth "go-boilerplate/internal/controller/stream/auth"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/infrastructure/eventlog"
	"go-boilerplate/internal/infrastructure/instancelease"
	"go-boilerplate/internal/infrastructure/realtimesecret"
	"go-boilerplate/internal/infrastructure/streamticket"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

// realtimeModule は、Realtime Delivery の store（EventLog / StreamTicket / InstanceLease / SecretGenerator）と
// 機構側 usecase（CursorValidator / TicketIssuer / TicketVerifier）、StreamTicket securityScheme の認証器を提供し、
// SSE の stream handler を Echo に登録する fx.Module です。handler が要る stream.Streamer は Phase 6 が提供します。
// 設計正本（docs/design/realtime-delivery.md §3.1）のとおり、feature の realtime adapter が 1 つ以上あるときに
// だけ app graph へ組み込みます。まだ無いので InfrastructureModule() には含めず、graph の検証だけを持ちます。
func realtimeModule() fx.Option {
	return fx.Module("realtime",
		fx.Provide(
			provideRealtimeClient,
			provideEventLogStore,
			provideStreamTicketStore,
			provideInstanceLeaseStore,
			provideRealtimeSecretGenerator,
			provideCursorValidator,
			provideTicketIssuer,
			provideTicketVerifier,
			fx.Annotate(provideStreamTicketScheme, fx.ResultTags(`group:"`+oapiauth.SchemeGroup+`"`)),
		),
		fx.Invoke(
			stream.BindHandler,
		),
	)
}

// provideRealtimeClient は、3 つの store が共有する DynamoDB クライアントを組み立てます。
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

func provideEventLogStore(c *awsdynamodb.Client, cfg *config.RealtimeConfig, tf observability.TracerFactory) rt.EventLogStore {
	return eventlog.New(c, cfg.EventLogTable(), tf)
}

func provideStreamTicketStore(c *awsdynamodb.Client, cfg *config.RealtimeConfig, tf observability.TracerFactory) rt.StreamTicketStore {
	return streamticket.New(c, cfg.StreamTicketTable(), tf)
}

func provideInstanceLeaseStore(c *awsdynamodb.Client, cfg *config.RealtimeConfig, tf observability.TracerFactory) rt.InstanceLeaseStore {
	return instancelease.New(c, cfg.InstanceLeaseTable(), tf)
}

func provideRealtimeSecretGenerator() rt.SecretGenerator {
	return realtimesecret.New()
}

func provideCursorValidator(log rt.EventLogStore, clk clock.Clock, tf observability.TracerFactory) ucrealtime.CursorValidator {
	return ucrealtime.NewCursorValidator(log, clk, tf)
}

func provideTicketIssuer(
	store rt.StreamTicketStore, secrets rt.SecretGenerator, clk clock.Clock, tf observability.TracerFactory,
) ucrealtime.TicketIssuer {
	return ucrealtime.NewTicketIssuer(store, secrets, clk, tf)
}

func provideTicketVerifier(store rt.StreamTicketStore, clk clock.Clock, tf observability.TracerFactory) ucrealtime.TicketVerifier {
	return ucrealtime.NewTicketVerifier(store, clk, tf)
}

// provideStreamTicketScheme は、query の stream ticket を検証する認証器を oapi/auth の scheme group へ出します。
func provideStreamTicketScheme(verifier ucrealtime.TicketVerifier) oapiauth.SchemeAuthenticator {
	return streamauth.New(verifier)
}
