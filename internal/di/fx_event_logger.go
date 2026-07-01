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
	return &fxEventLogger{logger: logger.Named("Fx").CallerSkip(callSkip)}
}

func (f *fxEventLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuted:
		f.record(e.Err,
			"OnStart hook failed",
			[]*logging.Field{logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName)},
			f.logger.Debug, "OnStart hook executed",
			logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName),
			logging.DurationMs("runtime_ms", e.Runtime))
	case *fxevent.OnStopExecuted:
		f.record(e.Err,
			"OnStop hook failed",
			[]*logging.Field{logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName)},
			f.logger.Debug, "OnStop hook executed",
			logging.String("callee", e.FunctionName), logging.String("caller", e.CallerName),
			logging.DurationMs("runtime_ms", e.Runtime))
	case *fxevent.Supplied:
		f.record(e.Err,
			"Supply failed",
			[]*logging.Field{logging.String("type", e.TypeName), logging.String("module", e.ModuleName)},
			f.logger.Debug, "Supplied",
			logging.String("type", e.TypeName), logging.String("module", e.ModuleName))
	case *fxevent.Provided:
		f.record(e.Err,
			"Provide failed",
			[]*logging.Field{logging.String("constructor", e.ConstructorName), logging.String("module", e.ModuleName)},
			f.logger.Debug, "Provided",
			logging.String("constructor", e.ConstructorName), logging.String("module", e.ModuleName),
			logging.Strings("output_types", e.OutputTypeNames))
	case *fxevent.Invoked:
		f.record(e.Err,
			"Invoke failed",
			[]*logging.Field{logging.String("function", e.FunctionName), logging.String("module", e.ModuleName)},
			f.logger.Debug, "Invoked",
			logging.String("function", e.FunctionName), logging.String("module", e.ModuleName))
	case *fxevent.Replaced:
		f.record(e.Err,
			"Replace failed",
			[]*logging.Field{logging.String("module", e.ModuleName)},
			f.logger.Debug, "Replaced",
			logging.Strings("output_types", e.OutputTypeNames), logging.String("module", e.ModuleName))
	case *fxevent.Decorated:
		f.record(e.Err,
			"Decorate failed",
			[]*logging.Field{logging.String("decorator", e.DecoratorName), logging.String("module", e.ModuleName)},
			f.logger.Debug, "Decorated",
			logging.String("decorator", e.DecoratorName), logging.String("module", e.ModuleName),
			logging.Strings("output_types", e.OutputTypeNames))
	case *fxevent.LoggerInitialized:
		f.record(e.Err,
			"Logger initialization failed", nil,
			f.logger.Debug, "Custom logger initialized",
			logging.String("constructor", e.ConstructorName))
	case *fxevent.Started:
		f.record(e.Err,
			"Application failed to start", nil,
			f.logger.Info, "Application started")
	case *fxevent.Stopped:
		f.record(e.Err,
			"Application failed to stop", nil,
			f.logger.Info, "Application stopped")
	case *fxevent.RollingBack:
		f.logger.Error("start failed, rolling back", logging.Error(logging.ErrorKey, e.StartErr))
	case *fxevent.RolledBack:
		f.record(e.Err,
			"Rollback failed", nil,
			f.logger.Debug, "Application rolled back")
	}
}

// record は fx イベントの成否に応じてログを記録します。
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
