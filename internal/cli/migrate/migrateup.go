package migrate

import (
	"strconv"

	"boilerplate-go/internal/logging"
	"boilerplate-go/pkg/xerrors"

	"github.com/spf13/cobra"

	"go.uber.org/zap"

	"github.com/golang-migrate/migrate/v4"
	// postgres driver for golang-migrate (required for runtime registration)
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// NewMigrateUpCommand は、DBのマイグレーションを上げるためのコマンドを生成します。
func NewMigrateUpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate-up [target_version]",
		Short: "database/migrationsのDDL適用をアップグレードします。",
		Long:  "このコマンドは、指定があれば特定バージョンまでUp操作を行います。引数なしなら全てUpします。",
		Args:  cobra.MaximumNArgs(1),
		RunE:  migrateUpRun,
	}
}

// migrateUpRun は、マイグレーションをアップデートするための実行関数です。
func migrateUpRun(_ *cobra.Command, args []string) error {
	logger := logging.NewProductionLogger()
	errors := xerrors.New()

	m, err := buildMigrateInstance()
	if err != nil {
		logger.Panic("failed to create migrate instance", zap.Error(err))
	}

	if len(args) == 0 {
		// 引数なしなら全てのマイグレーションをアップ
		logger.Info("running full migration up")
		if err := executeMigrateFullUp(m, errors); err != nil {
			logger.Panic("migration failed", zap.Error(err))
		}
	} else {
		// 引数がある場合は指定されたバージョンまでアップグレード
		logger.Info("running migration up to version", zap.String("steps", args[0]))
		if err := executeMigrateStepsUpFromArgs(m, args); err != nil {
			logger.Panic("migration to version failed", zap.Error(err))
		}
	}
	logger.Info("✅ migration completed")

	return nil
}

// executeMigrateFullUp は、マイグレーションを全てアップグレードします。
func executeMigrateFullUp(m *migrate.Migrate, errors xerrors.Errors) error {
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// executeMigrateStepsUpFromArgs は、指定されたステップ数だけマイグレーションをアップグレードします。
func executeMigrateStepsUpFromArgs(m *migrate.Migrate, args []string) error {
	steps, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}
	return m.Steps(int(steps))
}
