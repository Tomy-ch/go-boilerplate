package di

import (
	"context"
	"testing"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

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

func Test_fxEventLogger_LogEvent(t *testing.T) {
	t.Parallel()

	boom := xerrors.New("boom")

	// assertLogged は、指定イベントが期待メッセージ・レベルで 1 件だけ記録されることを検証する。
	assertLogged := func(t *testing.T, event fxevent.Event, wantMsg, wantLevel string) {
		t.Helper()

		obs, logs := logging.NewObservedTestLogger(t)
		NewFxEventLogger(obs).LogEvent(event)

		entries := logs.FilterMessage(wantMsg).All()
		require.Len(t, entries, 1)
		assert.Equal(t, wantLevel, entries[0].Level.String())
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OnStart失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.OnStartExecuted{FunctionName: "f", CallerName: "c", Err: boom}, "OnStart hook failed", "error")
		})

		t.Run("OnStart成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.OnStartExecuted{FunctionName: "f", CallerName: "c"}, "OnStart hook executed", "debug")
		})

		t.Run("OnStop失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.OnStopExecuted{FunctionName: "f", Err: boom}, "OnStop hook failed", "error")
		})

		t.Run("OnStop成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.OnStopExecuted{FunctionName: "f"}, "OnStop hook executed", "debug")
		})

		t.Run("Supply失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Supplied{TypeName: "T", Err: boom}, "Supply failed", "error")
		})

		t.Run("Supply成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Supplied{TypeName: "T", ModuleName: "m"}, "Supplied", "debug")
		})

		t.Run("Provide失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Provided{ConstructorName: "ctor", Err: boom}, "Provide failed", "error")
		})

		t.Run("Provide成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Provided{ConstructorName: "ctor", ModuleName: "m", OutputTypeNames: []string{"T"}}, "Provided", "debug")
		})

		t.Run("Invoke失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Invoked{FunctionName: "f", Err: boom}, "Invoke failed", "error")
		})

		t.Run("Invoke成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Invoked{FunctionName: "f", ModuleName: "m"}, "Invoked", "debug")
		})

		t.Run("Replace失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Replaced{Err: boom}, "Replace failed", "error")
		})

		t.Run("Replace成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Replaced{OutputTypeNames: []string{"T"}, ModuleName: "m"}, "Replaced", "debug")
		})

		t.Run("Decorate失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Decorated{DecoratorName: "dec", Err: boom}, "Decorate failed", "error")
		})

		t.Run("Decorate成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Decorated{DecoratorName: "dec", ModuleName: "m", OutputTypeNames: []string{"T"}}, "Decorated", "debug")
		})

		t.Run("LoggerInitialized失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.LoggerInitialized{Err: boom}, "Logger initialization failed", "error")
		})

		t.Run("LoggerInitialized成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.LoggerInitialized{ConstructorName: "ctor"}, "Custom logger initialized", "debug")
		})

		t.Run("Started失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Started{Err: boom}, "Application failed to start", "error")
		})

		t.Run("Started成功はInfoで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Started{}, "Application started", "info")
		})

		t.Run("Stopped失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Stopped{Err: boom}, "Application failed to stop", "error")
		})

		t.Run("Stopped成功はInfoで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.Stopped{}, "Application stopped", "info")
		})

		t.Run("RollingBackは起動失敗をErrorで記録する", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.RollingBack{StartErr: boom}, "start failed, rolling back", "error")
		})

		t.Run("RolledBack失敗はErrorで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.RolledBack{Err: boom}, "Rollback failed", "error")
		})

		t.Run("RolledBack成功はDebugで記録される", func(t *testing.T) {
			t.Parallel()
			assertLogged(t, &fxevent.RolledBack{}, "Application rolled back", "debug")
		})

		t.Run("対象外イベントは無視される", func(t *testing.T) {
			t.Parallel()

			obs, logs := logging.NewObservedTestLogger(t)
			NewFxEventLogger(obs).LogEvent(&fxevent.Invoking{FunctionName: "f"})

			assert.Equal(t, 0, logs.Len())
		})
	})
}

func Test_fxEventLogger_record(t *testing.T) {
	t.Parallel()

	boom := xerrors.New("boom")
	failFields := []*logging.Field{logging.String("phase", "provide")}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("errがnilの場合は成功用ロガーへ成功メッセージとフィールドを渡す", func(t *testing.T) {
			t.Parallel()

			obs, logs := logging.NewObservedTestLogger(t)
			f := &fxEventLogger{logger: obs}

			f.record(nil, "Provide failed", failFields, f.logger.Info, "Provided", logging.String("module", "m"))

			entries := logs.FilterMessage("Provided").All()
			require.Len(t, entries, 1)
			assert.Equal(t, "info", entries[0].Level.String())
			assert.Equal(t, map[string]any{"module": "m"}, entries[0].ContextMap()) // 失敗用フィールドは混ざらない
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("errが非nilの場合は成功用ロガーを呼ばず失敗フィールドとエラーを添えてErrorで記録する", func(t *testing.T) {
			t.Parallel()

			obs, logs := logging.NewObservedTestLogger(t)
			f := &fxEventLogger{logger: obs}
			okCalled := false
			logOK := func(context.Context, string, ...*logging.Field) { okCalled = true }

			f.record(boom, "Provide failed", failFields, logOK, "Provided", logging.String("module", "m"))

			assert.False(t, okCalled)
			entries := logs.FilterMessage("Provide failed").All()
			require.Len(t, entries, 1)
			assert.Equal(t, "error", entries[0].Level.String())
			ctxMap := entries[0].ContextMap()
			assert.Equal(t, "provide", ctxMap["phase"])
			assert.Contains(t, ctxMap, string(logging.ErrorKey))
		})
	})
}
