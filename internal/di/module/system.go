package module

import (
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/system"

	"go.uber.org/fx"
)

// SystemModule は、システム情報関連の依存関係を提供するfx.Moduleです。
func SystemModule() fx.Option {
	return fx.Module("system",
		fx.Provide(
			system.NewBuildInfo,
		),
		fx.Invoke(
			logBuildInfo,
		),
	)
}

// logBuildInfo は、起動時にアプリのビルド情報を1行ログへ出力します。
func logBuildInfo(logger logging.Logger, bi system.BuildInfo) {
	logger.Named("system.buildinfo").CallerSkip(callerSkipCount).Info(
		"application build info",
		logging.String("version", bi.Version()),
		logging.String("revision", bi.Revision()),
		logging.String("build_date", bi.BuildDate()),
	)
}
