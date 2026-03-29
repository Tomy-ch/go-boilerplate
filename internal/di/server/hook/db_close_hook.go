package hook

import (
	"context"

	"boilerplate-go/internal/di/lifecycle"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/logging"
)

// RegisterDBCloseHooks は、アプリケーションのシャットダウン時にデータベース接続を閉じるためのフックを登録します。
func RegisterDBCloseHooks(
	reg lifecycle.Registrar,
	db driver.DatabaseDriver,
	logger logging.Logger,
) {
	reg.RegisterStop(func(ctx context.Context) error {
		logger.Named("db.CloseHook").Info("Closing database connection")
		return db.Close()
	})
}
