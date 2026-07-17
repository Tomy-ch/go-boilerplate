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
	reg.RegisterStop(func(ctx context.Context) error {
		l := logger.Named("db.CloseHook")
		l.Info(ctx, "Closing database connection")
		if err := db.Close(); err != nil {
			l.Error(ctx, "failed to close database", logging.Error(logging.ErrorKey, err))
			return err
		}
		return nil
	})
}
