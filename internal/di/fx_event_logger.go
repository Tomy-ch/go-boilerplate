package di

import (
	"go-boilerplate/internal/logging"

	"go.uber.org/fx/fxevent"
)

// fxEventLogger は fx イベントを構造化ロガーへ流す fxevent.Logger 実装。
// 解決トレース（Provided / Invoked 等の成功イベント）は Debug、失敗は Error、
// 起動／停止の節目は Info で記録する。console 出力ではなく logging パッケージへ寄せることで、
// 起動失敗時に「何起因でどこまで解決が進んだか」を構造化ログとして追える（Debug 有効時）。
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
		if e.Err != nil {
			f.logger.Error("fx OnStart hook failed",
				logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName),
				logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx OnStart hook executed",
			logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName),
			logging.DurationMs("runtime_ms", e.Runtime))
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			f.logger.Error("fx OnStop hook failed",
				logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName),
				logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx OnStop hook executed",
			logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName),
			logging.DurationMs("runtime_ms", e.Runtime))
	case *fxevent.Supplied:
		if e.Err != nil {
			f.logger.Error("fx supply failed",
				logging.String("type", e.TypeName), logging.String("module", e.ModuleName),
				logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx supplied",
			logging.String("type", e.TypeName), logging.String("module", e.ModuleName))
	case *fxevent.Provided:
		if e.Err != nil {
			f.logger.Error("fx provide failed",
				logging.String("constructor", e.ConstructorName), logging.String("module", e.ModuleName),
				logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx provided",
			logging.String("constructor", e.ConstructorName), logging.String("module", e.ModuleName),
			logging.Strings("output_types", e.OutputTypeNames))
	case *fxevent.Invoked:
		if e.Err != nil {
			f.logger.Error("fx invoke failed",
				logging.String("function", e.FunctionName), logging.String("module", e.ModuleName),
				logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx invoked",
			logging.String("function", e.FunctionName), logging.String("module", e.ModuleName))
	case *fxevent.Replaced:
		if e.Err != nil {
			f.logger.Error("fx replace failed",
				logging.String("module", e.ModuleName), logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx replaced",
			logging.Strings("output_types", e.OutputTypeNames), logging.String("module", e.ModuleName))
	case *fxevent.Decorated:
		if e.Err != nil {
			f.logger.Error("fx decorate failed",
				logging.String("decorator", e.DecoratorName), logging.String("module", e.ModuleName),
				logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx decorated",
			logging.String("decorator", e.DecoratorName), logging.String("module", e.ModuleName),
			logging.Strings("output_types", e.OutputTypeNames))
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			f.logger.Error("fx logger initialization failed", logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx custom logger initialized", logging.String("constructor", e.ConstructorName))
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
		if e.Err != nil {
			f.logger.Error("fx rollback failed", logging.Error(logging.ErrorKey, e.Err))
			return
		}
		f.logger.Debug("fx rolled back")
	}
}
