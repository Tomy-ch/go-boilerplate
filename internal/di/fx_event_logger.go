package di

import (
	"go-boilerplate/internal/logging"

	"go.uber.org/fx/fxevent"
)

// fxEventLogger は fx イベントを構造化ロガーへ流す fxevent.Logger 実装。
// エラーと起動／停止の節目のみ記録し冗長な成功イベントは無視する。logging 層を
// フレームワーク非依存（depguard）に保つため fx を扱える di 層に置く。
type fxEventLogger struct {
	logger logging.Logger
}

// NewFxEventLogger は、fx.WithLogger に渡す fxevent.Logger を生成します。
func NewFxEventLogger(logger logging.Logger) fxevent.Logger {
	return &fxEventLogger{logger: logger.Named("fx")}
}

func (f *fxEventLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuted:
		f.errorIf(e.Err, "fx OnStart hook failed",
			logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName))
	case *fxevent.OnStopExecuted:
		f.errorIf(e.Err, "fx OnStop hook failed",
			logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName))
	case *fxevent.Supplied:
		f.errorIf(e.Err, "fx supply failed", logging.String("type", e.TypeName))
	case *fxevent.Provided:
		f.errorIf(e.Err, "fx provide failed", logging.String("constructor", e.ConstructorName))
	case *fxevent.Invoked:
		f.errorIf(e.Err, "fx invoke failed", logging.String("function", e.FunctionName))
	case *fxevent.Replaced:
		f.errorIf(e.Err, "fx replace failed")
	case *fxevent.Decorated:
		f.errorIf(e.Err, "fx decorate failed", logging.String("decorator", e.DecoratorName))
	case *fxevent.LoggerInitialized:
		f.errorIf(e.Err, "fx logger initialization failed")
	case *fxevent.Started:
		if e.Err != nil {
			f.logger.Error("fx application failed to start", logging.Error(logging.ErrorKey, e.Err))
		} else {
			f.logger.Info("fx application started")
		}
	case *fxevent.Stopped:
		if e.Err != nil {
			f.logger.Error("fx application failed to stop", logging.Error(logging.ErrorKey, e.Err))
		} else {
			f.logger.Info("fx application stopped")
		}
	case *fxevent.RollingBack:
		f.logger.Error("fx start failed, rolling back", logging.Error(logging.ErrorKey, e.StartErr))
	case *fxevent.RolledBack:
		f.errorIf(e.Err, "fx rollback failed")
	}
}

func (f *fxEventLogger) errorIf(err error, msg string, fields ...*logging.Field) {
	if err == nil {
		return
	}
	f.logger.Error(msg, append(fields, logging.Error(logging.ErrorKey, err))...)
}
