package di

import (
	"context"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/module"
	"go-boilerplate/internal/di/shutdowner"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/job"

	"go.uber.org/fx"
)

// callSkip は Named().CallerSkip() で飛ばすフレーム数。RunJob/RunWorker の匿名クロージャ + CallerSkip ラッパの計 2 段を補正する。関数構造を変えると見直しが必要。
const callSkip = 2

type (
	// StartFunc は、ジョブを起動して完了通知チャネルを返す関数の型です。返却チャネルはジョブ終了時に error（成功時 nil）を送信した後クローズされます。DI グラフの構築失敗および起動失敗のときは専用の閉じ済みチャネルで直接エラーを返します。
	StartFunc func(ctx context.Context, name string, args []string) <-chan error
	// StopFunc は、ジョブの停止関数の型を表します。
	StopFunc func(ctx context.Context) error
)

// failClosedChan は、err を 1 件送信したうえでクローズ済みのチャネルを返します。
// 共有の done チャネルは lifecycle hook の goroutine が送信・クローズを所有するため、
// start が自前で失敗を返す経路では二重送信・二重クローズを避けて専用チャネルを使います。
func failClosedChan(err error) <-chan error {
	ch := make(chan error, 1)
	ch <- err
	close(ch)

	return ch
}

// NewJobCore は、ジョブ実行用の fx.App を構成する fx.Option を返します。
func NewJobCore() fx.Option {
	return fx.Options(
		// Shutdowner Module
		shutdowner.Module(),
		// Lifecycle Module
		lifecycle.Module(),
		// Common Module
		module.ConfigModule(),
		module.LoggingModule(),
		module.ObservabilityModule(),
		module.DatabaseModule(),
		module.SystemModule(),
		// DDD Modules
		module.InfrastructureModule(),
		module.UsecaseModule(),
		// Job Module
		module.JobModule(),
	)
}

// RunJob は、ジョブ実行用の開始関数・停止関数を生成して返します。context／タイムアウトの制御は返した関数の呼出側が担います。
// grace（APP_SHUTDOWN_TIMEOUT）を fx.StopTimeout に設定し、停止時に fx 既定（15s）が
// 停止猶予より先に teardown を打ち切らないようにします（停止軸を grace に一本化）。
// DI グラフの構築に失敗した場合、返す start／stop はいずれもその構築エラーを返します。
func RunJob(grace time.Duration) (StartFunc, StopFunc) {
	var (
		state  job.State
		logger logging.Logger
		osCfg  *config.OperatingSystemConfig
	)

	app := fx.New(
		NewJobCore(),
		fx.StopTimeout(grace),
		fx.Populate(&state, &logger, &osCfg),
		fx.WithLogger(NewFxEventLogger),
	)

	start := func(ctx context.Context, name string, args []string) <-chan error {
		// fx.Populate は fx.New 時点の invoke なので、グラフ構築が失敗すると対象は nil のまま残る。
		// logger / osCfg を触る前に構築エラーを返し、nil 参照ではなく DI エラーをオペレータへ届ける。
		if err := app.Err(); err != nil {
			return failClosedChan(err)
		}

		l := logger.Named("job.RunJob").CallerSkip(callSkip)
		done := make(chan error, 1)
		state.Set(name, args, done)

		if err := app.Start(ctx); err != nil {
			fields := append(jobEventFields(logging.EventTypeStart, osCfg.TimeZone()),
				logging.String(logging.JobNameKey, name),
				logging.Strings(logging.JobArgsKey, args),
				logging.Error(logging.JobErrorKey, err),
			)
			l.Error(ctx, "failed to start job application", fields...)

			return failClosedChan(err)
		}

		fields := append(jobEventFields(logging.EventTypeStart, osCfg.TimeZone()),
			logging.String(logging.JobNameKey, name),
			logging.Strings(logging.JobArgsKey, args),
		)
		l.Info(ctx, "job started", fields...)

		return done
	}

	stop := func(ctx context.Context) error {
		// start と同じく、構築失敗時は nil の logger / osCfg を参照せずに構築エラーを返す。
		if err := app.Err(); err != nil {
			return err
		}

		l := logger.Named("job.RunJob").CallerSkip(callSkip)
		if err := app.Stop(ctx); err != nil {
			fields := append(jobEventFields(logging.EventTypeEnd, osCfg.TimeZone()),
				logging.Error(logging.JobErrorKey, err),
			)
			l.Error(ctx, "failed to stop job application", fields...)
			return err
		}

		l.Info(ctx, "job application finished", jobEventFields(logging.EventTypeEnd, osCfg.TimeZone())...)
		return nil
	}

	return start, stop
}

// jobEventFields は、ジョブ起動／停止ログ共通のイベントフィールド（種別・発生時刻・TZ）を返します。
func jobEventFields(eventType, tz string) []*logging.Field {
	return []*logging.Field{
		logging.String(logging.EventTypeKey, eventType),
		logging.Time(logging.EventAtKey, time.Now()),
		logging.String(logging.EventTzKey, tz),
	}
}
