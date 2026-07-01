package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/system"
	mock_system "go-boilerplate/internal/system/mock"
)

func TestSystemModule_ProvidesBuildInfo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで BuildInfo が提供される", func(t *testing.T) {
			t.Parallel()

			var bi system.BuildInfo

			app := fx.New(
				SystemModule(),
				// logBuildInfo の Invoke が logging.Logger を要求するため供給する。
				fx.Provide(func() logging.Logger { return logging.NewTestLogger(t) }),
				fx.Populate(&bi),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			require.NotNil(t, bi)
			// Methods should be callable
			_ = bi.Version()
			_ = bi.Revision()
			_ = bi.BuildDate()
			require.NoError(t, app.Stop(context.Background()))
		})
	})
}

func TestLogBuildInfo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("version / revision / build_date を付与して出力する", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.2.3")
			bi.EXPECT().Revision().Return("abc1234")
			bi.EXPECT().BuildDate().Return("2026-07-01T00:00:00Z")

			logBuildInfo(logger, bi)

			logs := observed.All()
			require.Len(t, logs, 1)

			entry := logs[0]
			assert.Equal(t, "application build info", entry.Message)
			assert.Equal(t, "system.buildinfo", entry.LoggerName)

			fields := entry.ContextMap()
			assert.Equal(t, "v1.2.3", fields["version"])
			assert.Equal(t, "abc1234", fields["revision"])
			assert.Equal(t, "2026-07-01T00:00:00Z", fields["build_date"])
		})
	})
}
