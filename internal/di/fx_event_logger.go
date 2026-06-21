package di

import (
	"go-boilerplate/internal/logging"

	"go.uber.org/fx/fxevent"
)

// fxEventLogger は fx イベントを構造化ロガーへ流す fxevent.Logger 実装。
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
		f.record(e.Err,
			"fx OnStart hook failed",
			[]*logging.Field{logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName)},
			f.logger.Debug, "fx OnStart hook executed",
			logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName),
			logging.DurationMs("runtime_ms", e.Runtime))
	case *fxevent.OnStopExecuted:
		f.record(e.Err,
			"fx OnStop hook failed",
			[]*logging.Field{logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName)},
			f.logger.Debug, "fx OnStop hook executed",
			logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName),
			logging.DurationMs("runtime_ms", e.Runtime))
	case *fxevent.Supplied:
		f.record(e.Err,
			"fx supply failed",
			[]*logging.Field{logging.String("type", e.TypeName), logging.String("module", e.ModuleName)},
			f.logger.Debug, "fx supplied",
			logging.String("type", e.TypeName), logging.String("module", e.ModuleName))
	case *fxevent.Provided:
		f.record(e.Err,
			"fx provide failed",
			[]*logging.Field{logging.String("constructor", e.ConstructorName), logging.String("module", e.ModuleName)},
			f.logger.Debug, "fx provided",
			logging.String("constructor", e.ConstructorName), logging.String("module", e.ModuleName),
			logging.Strings("output_types", e.OutputTypeNames))
	case *fxevent.Invoked:
		f.record(e.Err,
			"fx invoke failed",
			[]*logging.Field{logging.String("function", e.FunctionName), logging.String("module", e.ModuleName)},
			f.logger.Debug, "fx invoked",
			logging.String("function", e.FunctionName), logging.String("module", e.ModuleName))
	case *fxevent.Replaced:
		f.record(e.Err,
			"fx replace failed",
			[]*logging.Field{logging.String("module", e.ModuleName)},
			f.logger.Debug, "fx replaced",
			logging.Strings("output_types", e.OutputTypeNames), logging.String("module", e.ModuleName))
	case *fxevent.Decorated:
		f.record(e.Err,
			"fx decorate failed",
			[]*logging.Field{logging.String("decorator", e.DecoratorName), logging.String("module", e.ModuleName)},
			f.logger.Debug, "fx decorated",
			logging.String("decorator", e.DecoratorName), logging.String("module", e.ModuleName),
			logging.Strings("output_types", e.OutputTypeNames))
	case *fxevent.LoggerInitialized:
		f.record(e.Err,
			"fx logger initialization failed", nil,
			f.logger.Debug, "fx custom logger initialized",
			logging.String("constructor", e.ConstructorName))
	case *fxevent.Started:
		f.record(e.Err,
			"fx application failed to start", nil,
			f.logger.Info, "fx application started")
	case *fxevent.Stopped:
		f.record(e.Err,
			"fx application failed to stop", nil,
			f.logger.Info, "fx application stopped")
	case *fxevent.RollingBack:
		f.logger.Error("fx start failed, rolling back", logging.Error(logging.ErrorKey, e.StartErr))
	case *fxevent.RolledBack:
		f.record(e.Err,
			"fx rollback failed", nil,
			f.logger.Debug, "fx rolled back")
	}
}

// record は fx イベントの成否でログを振り分ける共通処理。
// 失敗時は failFields に error を添えて Error、成功時は logOK（Debug / Info）で okFields を記録する。
func (f *fxEventLogger) record(
	err error,
	failMsg string, failFields []*logging.Field,
	logOK func(string, ...*logging.Field), okMsg string, okFields ...*logging.Field,
) {
	if err != nil {
		f.logger.Error(failMsg, append(failFields, logging.Error(logging.ErrorKey, err))...)
		return
	}
	logOK(okMsg, okFields...)
}
