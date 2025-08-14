package migrate

import (
	"errors"
	"strconv"

	"boilerplate-go/internal/controller/middleware/logging"
	"boilerplate-go/pkg/safecast"

	"github.com/spf13/cobra"

	"go.uber.org/zap"

	"github.com/golang-migrate/migrate/v4"
	// postgres driver for golang-migrate (required for runtime registration)
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// NewMigrateDownCommand は、DBのマイグレーションを下げるためのコマンドを生成します。
func NewMigrateDownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate-down [target_version]",
		Short: "database/migrationsのDDL適用をダウングレードします。",
		Long:  "このコマンドは、指定があれば特定バージョンまでDown操作を行います。引数なしなら全てDownします。",
		Args:  cobra.MaximumNArgs(1),
		RunE:  migrateDownRun,
	}
}

// migrateDownRun は、マイグレーションをダウングレードするための実行関数です。
func migrateDownRun(_ *cobra.Command, args []string) error {
	logger := logging.NewProductionLogger()

	m, err := buildMigrateInstance()
	if err != nil {
		logger.Panic("failed to create migrate instance", zap.Error(err))
	}

	if len(args) == 0 {
		// 引数なしなら全てのマイグレーションをダウングレード
		logger.Info("running full migration down")
		err := executeMigrateFullDown(m)
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.Error("down migration failed", zap.Error(err))
		}
	} else {
		// 引数がある場合は指定されたバージョンまでダウングレード
		logger.Info("running migrate down steps", zap.String("steps", args[0]))
		if err := executeMigrateStepsDownFromArgs(m, args); err != nil {
			logger.Error("down migration steps failed", zap.Error(err))
		}
	}

	logger.Info("✅ migration down completed")
	return nil
}

// executeMigrateFullDown は、マイグレーションを全てダウングレードして、DBを初期状態に戻します。
func executeMigrateFullDown(m *migrate.Migrate) error {
	v, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}
	safeValue, err := safecast.UintToInt(v)
	if err != nil {
		return err
	}
	if dirty {
		if err := m.Force(safeValue); err != nil {
			return err
		}
	}
	return m.Down()
}

// executeMigrateStepsDownFromArgs は、指定されたステップ数だけマイグレーションをダウングレードします。
func executeMigrateStepsDownFromArgs(m *migrate.Migrate, args []string) error {
	steps, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}
	return m.Steps(int(-steps))
}
