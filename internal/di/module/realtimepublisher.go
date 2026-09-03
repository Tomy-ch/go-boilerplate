package module

import (
	"go.uber.org/fx"

	realtimeinfra "go-boilerplate/internal/infrastructure/realtime"
	"go-boilerplate/internal/observability"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// realtimePublisherModule は、realtime channel の relay が使う publish 先（EventLog へ append → wakeup を publish）を
// 提供する fx.Module です。EventLog の store と fan-out の構築子を realtimeModule() と共有するので、
// 両方を同じ graph に結線してはいけません（internal/di/module/README.md「Design Policy」）。
func realtimePublisherModule() fx.Option {
	return fx.Module("realtime_publisher",
		fx.Provide(
			observability.NewRealtimeMetrics,
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
	metrics *observability.RealtimeMetrics,
) publisherbndry.Publisher {
	return realtimeinfra.NewPublisher(log, f.clients, f.topicARN, tf, metrics)
}
