package module

import (
	outboxpublisher "go-boilerplate/internal/infrastructure/publisher"

	"go.uber.org/fx"
)

// outboxPublisherModule は、transactional outbox の publish 先を提供するfx.Moduleです。
// 実装の選択は判別子（OUTBOX_PUBLISHER）に基づき outboxpublisher.New が行います。
// 送信先の解決も実装ごとに New の内側で行うため、選ばれなかった実装の設定は要求されません。
func outboxPublisherModule() fx.Option {
	return fx.Module("outbox_publisher",
		fx.Provide(
			outboxpublisher.New,
		),
		provideHTTPClientProfiles(
			outboxpublisher.NewDownstreamProfile,
		),
		provideRequiredDownstreams(
			outboxpublisher.RequiredDownstream,
		),
	)
}
