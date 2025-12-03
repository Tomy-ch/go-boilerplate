package migrate

import (
	"errors"

	"boilerplate-go/internal/logging"
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
	cmd := &cobra.Command{
		Use:   "migrate-down [target_version]",
		Short: "database/migrations のDDLをダウングレードします（--version / --database指定可）。",
		Long: `database/migrations ディレクトリに存在するDDLマイグレーションを適用します。

引数なしの場合は全てのマイグレーションをDownします。
--version フラグを指定すると、そのバージョンまでDownします。
--database フラグを指定すると、対象のデータベース（例: local, test）を指定してDownを行います。`,
		RunE: migrateDownRun,
	}

	cmd.Flags().IntVar(&targetVersion, "version", 0, "filter VERSION")
	cmd.Flags().StringVar(&targetDatabase, "database", "", "filter DATABASE (e.g. local)")

	return cmd
}

// migrateDownRun は、マイグレーションをダウングレードするための実行関数です。
func migrateDownRun(_ *cobra.Command, _ []string) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	m, err := buildMigrateInstance(targetDatabase)
	if err != nil {
		logger.Panic("failed to create migrate instance", zap.Error(err))
	}

	if targetVersion == 0 {
		// 引数なしなら全てのマイグレーションをダウングレード
		logger.Info("running full migration down")
		err := executeMigrateFullDown(m)
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.Error("down migration failed", zap.Error(err))
		}
	} else {
		// 引数がある場合は指定されたバージョンまでダウングレード
		logger.Info("running migrate down steps", zap.Int("steps", targetVersion))
		if err := m.Steps(int(-targetVersion)); err != nil {
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
