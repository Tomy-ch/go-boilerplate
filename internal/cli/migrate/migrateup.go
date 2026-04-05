package migrate

import (
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/spf13/cobra"

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
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	// CLI オプションを反映した migrate インスタンスを組み立てます。
	m, err := buildMigrateInstance(targetDatabase)
	if err != nil {
		logger.Named("migrateUpRun.buildMigrateInstance").Error("failed to create migrate instance",
			logging.Error("buildMigrateInstance", err),
		)
		return err
	}

	if targetVersion == 0 {
		// バージョン未指定なら、未適用の migration を最後まで進めます。
		logger.Named("migrateUpRun").Info("running full migration up")
		if err := executeMigrateFullUp(m); err != nil {
			logger.Named("migrateUpRun.executeMigrateFullUp").Error("migration failed",
				logging.Error("executeMigrateFullUp", err),
			)
			return err
		}
	} else {
		// バージョン指定時は、現在位置から指定回数分だけ段階的に Up を進めます。
		logger.Named("migrateUpRun").Info("running migration up to version", logging.Int("steps", targetVersion))
		if err := m.Steps(targetVersion); err != nil {
			logger.Named("migrateUpRun.migrateUpSteps").Error("migration to version failed",
				logging.Error("migrateUpSteps", err),
			)
			return err
		}
	}
	logger.Named("migrateUpRun").Info("✅ migration completed")

	return nil
}

// executeMigrateFullUp は、マイグレーションを全てアップグレードします。
func executeMigrateFullUp(m *migrate.Migrate) error {
	// 既に最新であれば ErrNoChange になるため、成功扱いとして握りつぶします。
	if err := m.Up(); err != nil && !xerrors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
