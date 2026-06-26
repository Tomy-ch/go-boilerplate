package module

import (
	outboxpublisher "go-boilerplate/internal/infrastructure/publisher"
	userqs "go-boilerplate/internal/infrastructure/rdb/query_service/user" // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/prefecture"     // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/user"           // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/system_query/healthcheck"
	idempotencysq "go-boilerplate/internal/infrastructure/rdb/system_query/idempotency"
	outboxsq "go-boilerplate/internal/infrastructure/rdb/system_query/outbox"
	"go-boilerplate/internal/infrastructure/security"
	"go-boilerplate/internal/infrastructure/system"
	exchangerateext "go-boilerplate/internal/infrastructure/webapi/exchangerate" // sample-api:line

	"go.uber.org/fx"
)

// InfrastructureModule は、インフラストラクチャ層の依存関係を提供するfx.Moduleです。
func InfrastructureModule() fx.Option {
	return fx.Module("infrastructure",
		fx.Module("persistence",
			fx.Module("repository",
				fx.Provide(
					// sample-api:begin
					// サンプルのリポジトリ
					user.New,
					prefecture.New,
					// sample-api:end
				),
			),
			fx.Module("query_service",
				fx.Provide(
					// sample-api:begin
					// サンプルのクエリサービス
					userqs.New,
					// sample-api:end
				),
			),
			fx.Module("command_service",
				fx.Provide(
				// sample-api:begin
				// コマンドサービスは、このサンプルでは用意しませんが、必要に応じてここに追加します。
				// sample-api:end
				),
			),
			fx.Module("system_query",
				fx.Provide(
					healthcheck.New,
					idempotencysq.New,
					outboxsq.New,
				),
			),
		),
		clockModule(),
		httpClientModule(),
		webapiModule(),
		outboxPublisherModule(),
		fx.Module("security",
			fx.Provide(
				security.NewBcryptHasher,
			),
		),
	)
}

// clockModule は、時刻・待機関連の依存を提供するfx.Moduleです。
func clockModule() fx.Option {
	return fx.Module("clock",
		fx.Provide(
			system.NewClock,
			system.NewSleeper,
		),
	)
}

// webapiModule は、外部 Web API クライアント（gateway）を提供するfx.Moduleです。
func webapiModule() fx.Option {
	return fx.Module("webapi",
		fx.Provide(
			// サンプルの外部サービス gateway（DTO モード）
			exchangerateext.NewEndpoint, // sample-api:line
			exchangerateext.New,         // sample-api:line
		),
		provideHTTPClientProfiles(
			exchangerateext.NewDownstreamProfile, // sample-api:line
		),
	)
}

// outboxPublisherModule は、transactional outbox の publish 先（HTTP）を提供するfx.Moduleです。
func outboxPublisherModule() fx.Option {
	return fx.Module("outbox_publisher",
		fx.Provide(
			outboxpublisher.NewEndpoint,
			outboxpublisher.New,
		),
		provideHTTPClientProfiles(
			outboxpublisher.NewDownstreamProfile,
		),
	)
}
