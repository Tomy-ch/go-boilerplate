package migrate

import (
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
	cmd := &cobra.Command{
		Use:   "migrate-up [target_version]",
		Short: "database/migrations のDDLをアップグレードします（--version / --database指定可）。",
		Long: `database/migrations ディレクトリに存在するDDLマイグレーションを適用します。

引数なしの場合は全てのマイグレーションをUpします。
--version フラグを指定すると、そのバージョンまでUpします。
--database フラグを指定すると、対象のデータベース（例: local, test）を指定してUpを行います。`,
		RunE: migrateUpRun,
	}

	cmd.Flags().IntVar(&targetVersion, "version", 0, "filter VERSION")
	cmd.Flags().StringVar(&targetDatabase, "database", "", "filter DATABASE (e.g. local)")

	return cmd
}

// migrateUpRun は、マイグレーションをアップデートするための実行関数です。
func migrateUpRun(_ *cobra.Command, _ []string) error {
	logger := logging.NewProductionLogger()

	m, err := buildMigrateInstance(targetDatabase)
	if err != nil {
		logger.Panic("failed to create migrate instance", zap.Error(err))
	}

	if targetVersion == 0 {
		// 引数なしなら全てのマイグレーションをアップ
		logger.Info("running full migration up")
		if err := executeMigrateFullUp(m); err != nil {
			logger.Panic("migration failed", zap.Error(err))
		}
	} else {
		// 引数がある場合は指定されたバージョンまでアップグレード
		logger.Info("running migration up to version", zap.Int("steps", targetVersion))
		if err := m.Steps(targetVersion); err != nil {
			logger.Panic("migration to version failed", zap.Error(err))
		}
	}
	logger.Info("✅ migration completed")

	return nil
}

// executeMigrateFullUp は、マイグレーションを全てアップグレードします。
func executeMigrateFullUp(m *migrate.Migrate) error {
	if err := m.Up(); err != nil && !xerrors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
