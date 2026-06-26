package module

import (
	"go.uber.org/fx"
)

// InfrastructureModule は、インフラストラクチャ層の依存関係を集約して提供するfx.Moduleです。
// 各 concern（永続化 / clock / HTTP client / webapi gateway / outbox publisher / security）は
// 同パッケージ内の concern ごとのファイルにサブモジュールとして切り出し、ここで束ねます。
func InfrastructureModule() fx.Option {
	return fx.Module("infrastructure",
		persistenceModule(),
		clockModule(),
		httpClientModule(),
		webapiModule(),
		outboxPublisherModule(),
		securityModule(),
	)
}
