package hook

import (
	"context"

	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
)

// RegisterDBCloseHooks は、アプリケーションのシャットダウン時にデータベース接続を閉じるためのフックを登録します。
func RegisterDBCloseHooks(
	reg lifecycle.Registrar,
	db driver.DatabaseDriver,
	logger logging.Logger,
) {
	reg.RegisterStop(func(_ context.Context) error {
		logger.Named("db.CloseHook").Info("Closing database connection")
		return db.Close()
	})
}
