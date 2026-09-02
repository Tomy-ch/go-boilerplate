package module

import (
	"context"

	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/job/orphancleanup"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/infrastructure/instancelease"
	realtimeinfra "go-boilerplate/internal/infrastructure/realtime"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// RealtimeCleanupModule は、orphan cleanup ジョブを job profile へ登録する fx.Module です。
// ジョブの登録（group:"jobs"）まで自分で行うため、JobModule() は Realtime を知りません。
//
// Realtime の依存は graph に載せず、ジョブが選ばれたときに SweeperFactory の中で組み立てます
// （eager に載せてはならない理由は internal/di/module/README.md「Design Policy」）。
func RealtimeCleanupModule() fx.Option {
	return fx.Module("realtime-cleanup",
		fx.Provide(
			// 計装は MeterProvider だけを要求するので、Realtime の substrate を graph に載せません。
			observability.NewRealtimeMetrics,
			provideOrphanSweeperFactory,
		),
		provideJobs(orphancleanup.New),
	)
}

// provideOrphanSweeperFactory は、掃除役を実行時に組み立てる関数を返します。設定の検証と資格情報の解決も
// この関数の中で起きるため、ジョブが実行されない限り Realtime の設定は要求されません。
func provideOrphanSweeperFactory(
	cfg *config.RealtimeConfig,
	epCfg *config.EndpointConfig,
	outbound *observability.OutboundHTTPClient,
	clk clock.Clock,
	tf observability.TracerFactory,
) orphancleanup.SweeperFactory {
	return func(ctx context.Context) (ucrealtime.OrphanSweeper, error) {
		dynamo, err := dynamodbclient.New(ctx, dynamodbclient.Config{
			Endpoint:        epCfg.Realtime(),
			Region:          cfg.Region(),
			AccessKeyID:     cfg.AccessKeyID(),
			SecretAccessKey: cfg.SecretAccessKey(),
			HTTPClient:      outbound,
		})
		if err != nil {
			return nil, err
		}

		fanout, err := newRealtimeFanout(ctx, cfg.Topic(), realtimeinfra.ClientConfig{
			Endpoint:        epCfg.RealtimePubSub(),
			Region:          cfg.Region(),
			AccessKeyID:     cfg.AccessKeyID(),
			SecretAccessKey: cfg.SecretAccessKey(),
			HTTPClient:      outbound,
		})
		if err != nil {
			return nil, err
		}

		// 引き受けの主体を表す識別子は実行ごとに採番します。同時に走った掃除役どうしを区別できればよく、
		// 実行をまたいで安定である必要はありません。
		owner, err := uuid.New()
		if err != nil {
			return nil, xerrors.Wrap(err, "generate orphan cleanup owner id")
		}

		reclaimer := realtimeinfra.NewOrphanReclaimer(
			fanout.clients,
			realtimeinfra.SubscriptionTarget{TopicARN: fanout.topicARN, QueuePrefix: cfg.QueuePrefix()},
			tf,
		)

		return ucrealtime.NewOrphanSweeper(
			instancelease.New(dynamo, cfg.InstanceLeaseTable(), tf), reclaimer, owner.String(), clk, tf,
		), nil
	}
}
