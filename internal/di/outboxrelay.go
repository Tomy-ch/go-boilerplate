package di

import (
	"context"
	"time"

	"go.uber.org/fx"

	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/module"
	outboxuc "go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/uuid"
)

// outboxReplayStopTimeout は、replay 実行後に fx.App を停止する際の猶予です。
const outboxReplayStopTimeout = 30 * time.Second

// outboxRelayCommonOptions は、relay / replay が共有する共通モジュール群を返します。
func outboxRelayCommonOptions() []fx.Option {
	return []fx.Option{
		lifecycle.Module(),
		module.ConfigModule(),
		module.LoggingModule(),
		module.ObservabilityModule(),
		module.DatabaseModule(),
		module.SystemModule(),
		module.InfrastructureModule(),
		module.UsecaseModule(),
	}
}

// NewOutboxRelayCore は、relay 常駐プロセス用の fx.App を構成する fx.Option を返します。
func NewOutboxRelayCore() fx.Option {
	return fx.Options(append(outboxRelayCommonOptions(), module.OutboxRelayModule())...)
}

// NewOutboxRelayApp は、relay 常駐プロセス用の fx.App を生成します。
// OutboxRelayModule の OnStart フックが poll ループを起動します。
func NewOutboxRelayApp() *fx.App {
	return fx.New(NewOutboxRelayCore(), fx.WithLogger(NewFxEventLogger))
}

// RunOutboxReplay は、dead 状態の outbox 行を pending へ戻すワンショット実行を行います。
// poll ループを持たない（OutboxRelayModule を含まない）一時的な fx.App で ReplayUsecase を解決します。
func RunOutboxReplay(ctx context.Context, messageID *uuid.UUID) (int64, error) {
	var replay outboxuc.ReplayUsecase

	opts := append(outboxRelayCommonOptions(), fx.Populate(&replay), fx.WithLogger(NewFxEventLogger))
	app := fx.New(opts...)

	if err := app.Start(ctx); err != nil {
		return 0, err
	}

	result, runErr := replay.ReplayDead(ctx, messageID)

	stopCtx, cancel := context.WithTimeout(context.Background(), outboxReplayStopTimeout)
	defer cancel()
	stopErr := app.Stop(stopCtx) //nolint:contextcheck // 停止用 context は意図的に ctx を継承しない

	// replay 成功時は OnStop の失敗（DB プール Close 等）をオペレータへ可視化するため返す。
	// replay 自体が失敗していればそちらを優先する。
	if runErr != nil {
		return result, runErr
	}
	return result, stopErr
}
