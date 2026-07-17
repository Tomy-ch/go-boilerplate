package module

import (
	outboxpublisher "go-boilerplate/internal/infrastructure/publisher"

	"go.uber.org/fx"
)

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
		provideRequiredDownstreams(
			outboxpublisher.RequiredDownstream,
		),
	)
}
