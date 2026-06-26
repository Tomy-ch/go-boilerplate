package module

import (
	"go.uber.org/fx"
)

// InfrastructureModule は、全プロセス共通のインフラストラクチャ層の依存関係を集約して提供するfx.Moduleです。
// 各 concern（永続化 / clock / HTTP client / webapi gateway / security）は
// 同パッケージ内の concern ごとのファイルにサブモジュールとして切り出し、ここで束ねます。
// outbox publisher は relay 専用のため含めず、OutboxRelayModule 側で提供します。
func InfrastructureModule() fx.Option {
	return fx.Module("infrastructure",
		persistenceModule(),
		clockModule(),
		httpClientModule(),
		webapiModule(),
		securityModule(),
	)
}
