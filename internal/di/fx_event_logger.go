package di

import (
	"go-boilerplate/internal/logging"

	"go.uber.org/fx/fxevent"
)

// fxEventLogger は、fx のライフサイクルイベントを本プロジェクトの構造化ロガーへ流す
// fxevent.Logger 実装です。fx.WithLogger に渡すことで、既定の ConsoleLogger
// （非構造化・stderr）を置換します。
//
// エラーと起動／停止の節目のみ記録し、冗長な Provided／Supplied／Invoking 等の
// 成功イベントは無視してログノイズを抑えます。fx を扱える di 層に置くことで、
// logging 層をフレームワーク非依存（depguard）に保ったまま構造化を実現します。
type fxEventLogger struct {
	logger logging.Logger
}

// NewFxEventLogger は、fx.WithLogger に渡す fxevent.Logger を生成します。
func NewFxEventLogger(logger logging.Logger) fxevent.Logger {
	return &fxEventLogger{logger: logger.Named("fx")}
}

// LogEvent は、fx のライフサイクルイベントを構造化ログへ変換します。
func (f *fxEventLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			f.logger.Error("fx OnStart hook failed",
				logging.String("callee", e.FunctionName),
				logging.String("caller", e.CallerName),
				logging.Error(logging.ErrorKey, e.Err),
			)
		}
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			f.logger.Error("fx OnStop hook failed",
				logging.String("callee", e.FunctionName),
				logging.String("caller", e.CallerName),
				logging.Error(logging.ErrorKey, e.Err),
			)
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			f.logger.Error("fx supply failed",
				logging.String("type", e.TypeName),
				logging.Error(logging.ErrorKey, e.Err),
			)
		}
	case *fxevent.Provided:
		if e.Err != nil {
			f.logger.Error("fx provide failed",
				logging.String("constructor", e.ConstructorName),
				logging.Error(logging.ErrorKey, e.Err),
			)
		}
	case *fxevent.Invoked:
		if e.Err != nil {
			f.logger.Error("fx invoke failed",
				logging.String("function", e.FunctionName),
				logging.Error(logging.ErrorKey, e.Err),
			)
		}
	case *fxevent.Started:
		if e.Err != nil {
			f.logger.Error("fx application failed to start", logging.Error(logging.ErrorKey, e.Err))
		} else {
			f.logger.Info("fx application started")
		}
	case *fxevent.Stopped:
		if e.Err != nil {
			f.logger.Error("fx application failed to stop", logging.Error(logging.ErrorKey, e.Err))
		}
	case *fxevent.RolledBack:
		f.logger.Error("fx start rolled back", logging.Error(logging.ErrorKey, e.Err))
	}
}
