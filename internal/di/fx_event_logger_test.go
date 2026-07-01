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

	// assertLogged は、指定イベントが期待メッセージ・レベルで 1 件だけ記録されることを検証する。
	assertLogged := func(t *testing.T, event fxevent.Event, wantMsg, wantLevel string) {
		t.Helper()

		obs, logs := logging.NewObservedTestLogger(t)
		NewFxEventLogger(obs).LogEvent(event)

		entries := logs.FilterMessage(wantMsg).All()
		require.Len(t, entries, 1)
		assert.Equal(t, wantLevel, entries[0].Level.String())
	}

	t.Run("OnStart失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.OnStartExecuted{FunctionName: "f", CallerName: "c", Err: boom}, "fx OnStart hook failed", "error")
	})

	t.Run("OnStart成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.OnStartExecuted{FunctionName: "f", CallerName: "c"}, "fx OnStart hook executed", "debug")
	})

	t.Run("OnStop失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.OnStopExecuted{FunctionName: "f", Err: boom}, "fx OnStop hook failed", "error")
	})

	t.Run("OnStop成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.OnStopExecuted{FunctionName: "f"}, "fx OnStop hook executed", "debug")
	})

	t.Run("Supply失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Supplied{TypeName: "T", Err: boom}, "fx supply failed", "error")
	})

	t.Run("Supply成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Supplied{TypeName: "T", ModuleName: "m"}, "fx supplied", "debug")
	})

	t.Run("Provide失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Provided{ConstructorName: "ctor", Err: boom}, "fx provide failed", "error")
	})

	t.Run("Provide成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Provided{ConstructorName: "ctor", ModuleName: "m", OutputTypeNames: []string{"T"}}, "fx provided", "debug")
	})

	t.Run("Invoke失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Invoked{FunctionName: "f", Err: boom}, "fx invoke failed", "error")
	})

	t.Run("Invoke成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Invoked{FunctionName: "f", ModuleName: "m"}, "fx invoked", "debug")
	})

	t.Run("Replace失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Replaced{Err: boom}, "fx replace failed", "error")
	})

	t.Run("Replace成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Replaced{OutputTypeNames: []string{"T"}, ModuleName: "m"}, "fx replaced", "debug")
	})

	t.Run("Decorate失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Decorated{DecoratorName: "dec", Err: boom}, "fx decorate failed", "error")
	})

	t.Run("Decorate成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Decorated{DecoratorName: "dec", ModuleName: "m", OutputTypeNames: []string{"T"}}, "fx decorated", "debug")
	})

	t.Run("LoggerInitialized失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.LoggerInitialized{Err: boom}, "fx logger initialization failed", "error")
	})

	t.Run("LoggerInitialized成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.LoggerInitialized{ConstructorName: "ctor"}, "fx custom logger initialized", "debug")
	})

	t.Run("Started失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Started{Err: boom}, "fx application failed to start", "error")
	})

	t.Run("Started成功はInfoで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Started{}, "fx application started", "info")
	})

	t.Run("Stopped失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Stopped{Err: boom}, "fx application failed to stop", "error")
	})

	t.Run("Stopped成功はInfoで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.Stopped{}, "fx application stopped", "info")
	})

	t.Run("RollingBackは起動失敗をErrorで記録する", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.RollingBack{StartErr: boom}, "fx start failed, rolling back", "error")
	})

	t.Run("RolledBack失敗はErrorで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.RolledBack{Err: boom}, "fx rollback failed", "error")
	})

	t.Run("RolledBack成功はDebugで記録される", func(t *testing.T) {
		t.Parallel()
		assertLogged(t, &fxevent.RolledBack{}, "fx rolled back", "debug")
	})

	t.Run("対象外イベントは無視される", func(t *testing.T) {
		t.Parallel()

		obs, logs := logging.NewObservedTestLogger(t)
		NewFxEventLogger(obs).LogEvent(&fxevent.Invoking{FunctionName: "f"})

		assert.Equal(t, 0, logs.Len())
	})
}
