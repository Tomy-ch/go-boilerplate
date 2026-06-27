package di

import (
	"context"
	"errors"
	"time"

	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/module"
	outboxuc "go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/uuid"
)

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
// grace（APP_SHUTDOWN_TIMEOUT）を fx.StopTimeout に設定し、fx 既定（15s）が停止猶予より
// 先に teardown を打ち切らないようにします（停止軸を grace に一本化）。
func NewOutboxRelayApp(grace time.Duration) *fx.App {
	return fx.New(NewOutboxRelayCore(), fx.StopTimeout(grace), fx.WithLogger(NewFxEventLogger))
}

// RunOutboxReplay は、dead 状態の outbox 行を pending へ戻すワンショット実行を行い、
// 書き換えた行数と発生したエラーを返します。messageID が nil の場合は全 dead 行を対象とします。
// 停止処理のエラーは runErr と errors.Join で結合して返します。
func RunOutboxReplay(ctx context.Context, messageID *uuid.UUID) (int64, error) {
	var (
		replay outboxuc.ReplayUsecase
		appCfg *config.ApplicationConfig
	)

	opts := append(outboxRelayCommonOptions(),
		fx.Populate(&replay, &appCfg), fx.WithLogger(NewFxEventLogger))
	app := fx.New(opts...)

	if err := app.Start(ctx); err != nil {
		return 0, err
	}

	result, runErr := replay.ReplayDead(ctx, messageID)

	// 停止猶予は APP_SHUTDOWN_TIMEOUT に一本化（独立した magic number を持たない）。
	stopCtx, cancel := context.WithTimeout(context.Background(), appCfg.ShutdownTimeout())
	defer cancel()
	stopErr := app.Stop(stopCtx) //nolint:contextcheck // 停止用 context は意図的に ctx を継承しない

	// replay 失敗と OnStop 失敗（DB プール Close 等）の双方をオペレータへ可視化する。
	return result, errors.Join(runErr, stopErr)
}
