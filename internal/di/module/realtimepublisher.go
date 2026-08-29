package module

import (
	"go.uber.org/fx"

	realtimeinfra "go-boilerplate/internal/infrastructure/realtime"
	"go-boilerplate/internal/observability"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// realtimePublisherModule は、realtime channel の relay が使う publish 先（EventLog へ append → wakeup を publish）を
// 提供する fx.Module です。OutboxRelayModule が channel=realtime のときだけ束ねます。
// EventLog の store と fan-out のクライアントは realtimeModule() と同じ構築子を共有しますが、stream handler の
// 登録は relay process に無関係なので realtimeModule() 自体は束ねません（serve と relay は別 process）。
func realtimePublisherModule() fx.Option {
	return fx.Module("realtime_publisher",
		fx.Provide(
			provideRealtimeClient,
			provideEventLogStore,
			provideRealtimeFanout,
			provideRealtimePublisher,
		),
	)
}

func provideRealtimePublisher(
	log rt.EventLogStore,
	f realtimeFanout,
	tf observability.TracerFactory,
) publisherbndry.Publisher {
	return realtimeinfra.NewPublisher(log, f.clients, f.topicARN, tf)
}
