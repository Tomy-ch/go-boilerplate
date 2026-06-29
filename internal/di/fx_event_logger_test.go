package di

import (
	"errors"
	"testing"

	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxevent"
)

func TestNewFxEventLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのfxevent.Loggerを返す", func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, NewFxEventLogger(logging.NewTestLogger(t)))
		})
	})
}

func TestFxEventLogger_LogEvent(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")

	cases := []struct {
		name      string
		event     fxevent.Event
		wantMsg   string // "" のときはログが出ないことを期待
		wantLevel string
	}{
		{"OnStart失敗はErrorで記録される", &fxevent.OnStartExecuted{FunctionName: "f", CallerName: "c", Err: boom}, "fx OnStart hook failed", "error"},
		{"OnStart成功はDebugで記録される", &fxevent.OnStartExecuted{FunctionName: "f", CallerName: "c"}, "fx OnStart hook executed", "debug"},
		{"OnStop失敗はErrorで記録される", &fxevent.OnStopExecuted{FunctionName: "f", Err: boom}, "fx OnStop hook failed", "error"},
		{"OnStop成功はDebugで記録される", &fxevent.OnStopExecuted{FunctionName: "f"}, "fx OnStop hook executed", "debug"},
		{"Supply失敗はErrorで記録される", &fxevent.Supplied{TypeName: "T", Err: boom}, "fx supply failed", "error"},
		{"Supply成功はDebugで記録される", &fxevent.Supplied{TypeName: "T", ModuleName: "m"}, "fx supplied", "debug"},
		{"Provide失敗はErrorで記録される", &fxevent.Provided{ConstructorName: "ctor", Err: boom}, "fx provide failed", "error"},
		{
			"Provide成功はDebugで記録される",
			&fxevent.Provided{ConstructorName: "ctor", ModuleName: "m", OutputTypeNames: []string{"T"}},
			"fx provided",
			"debug",
		},
		{"Invoke失敗はErrorで記録される", &fxevent.Invoked{FunctionName: "f", Err: boom}, "fx invoke failed", "error"},
		{"Invoke成功はDebugで記録される", &fxevent.Invoked{FunctionName: "f", ModuleName: "m"}, "fx invoked", "debug"},
		{"Replace失敗はErrorで記録される", &fxevent.Replaced{Err: boom}, "fx replace failed", "error"},
		{"Replace成功はDebugで記録される", &fxevent.Replaced{OutputTypeNames: []string{"T"}, ModuleName: "m"}, "fx replaced", "debug"},
		{"Decorate失敗はErrorで記録される", &fxevent.Decorated{DecoratorName: "dec", Err: boom}, "fx decorate failed", "error"},
		{
			"Decorate成功はDebugで記録される",
			&fxevent.Decorated{DecoratorName: "dec", ModuleName: "m", OutputTypeNames: []string{"T"}},
			"fx decorated",
			"debug",
		},
		{"LoggerInitialized失敗はErrorで記録される", &fxevent.LoggerInitialized{Err: boom}, "fx logger initialization failed", "error"},
		{"LoggerInitialized成功はDebugで記録される", &fxevent.LoggerInitialized{ConstructorName: "ctor"}, "fx custom logger initialized", "debug"},
		{"Started失敗はErrorで記録される", &fxevent.Started{Err: boom}, "fx application failed to start", "error"},
		{"Started成功はInfoで記録される", &fxevent.Started{}, "fx application started", "info"},
		{"Stopped失敗はErrorで記録される", &fxevent.Stopped{Err: boom}, "fx application failed to stop", "error"},
		{"Stopped成功はInfoで記録される", &fxevent.Stopped{}, "fx application stopped", "info"},
		{"RollingBackは起動失敗をErrorで記録する", &fxevent.RollingBack{StartErr: boom}, "fx start failed, rolling back", "error"},
		{"RolledBack失敗はErrorで記録される", &fxevent.RolledBack{Err: boom}, "fx rollback failed", "error"},
		{"RolledBack成功はDebugで記録される", &fxevent.RolledBack{}, "fx rolled back", "debug"},
		{"対象外イベントは無視される", &fxevent.Invoking{FunctionName: "f"}, "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obs, logs := logging.NewObservedTestLogger(t)
			NewFxEventLogger(obs).LogEvent(tc.event)

			if tc.wantMsg == "" {
				assert.Equal(t, 0, logs.Len())
				return
			}

			entries := logs.FilterMessage(tc.wantMsg).All()
			require.Len(t, entries, 1)
			assert.Equal(t, tc.wantLevel, entries[0].Level.String())
		})
	}
}
