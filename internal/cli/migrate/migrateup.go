package migrate

import (
	"errors"
	"fmt"

	"go-boilerplate/internal/logging"

	"github.com/spf13/cobra"

	"github.com/golang-migrate/migrate/v4"
	// postgres driver for golang-migrate (required for runtime registration)
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// NewMigrateUpCommand は、DBのマイグレーションを上げるためのコマンドを生成します。
func NewMigrateUpCommand() *cobra.Command {
	var (
		steps    int
		database string
	)

	cmd := &cobra.Command{
		Use:   "migrate-up",
		Short: "database/migrations のDDLをアップグレードします（--steps / --database指定可）。",
		Long: `database/migrations ディレクトリに存在するDDLマイグレーションを適用します。

--steps を指定しない場合（0）は、未適用のマイグレーションを全て Up します。
--steps に正の整数を指定すると、現在位置からその段数だけ Up します。
--database フラグを指定すると、対象のデータベース（例: local, test）に対して Up を行います。`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return migrateUpRun(steps, database, buildMigrateInstance)
		},
	}

	cmd.Flags().IntVar(&steps, "steps", 0, "現在位置から Up する段数（0 で全件、正の整数のみ）")
	cmd.Flags().StringVar(&database, "database", "", "対象データベース（例: local）")

	return cmd
}

// migrateUpRun は、マイグレーションをアップグレードするための実行関数です。
func migrateUpRun(steps int, database string, newMigrator migratorFactory) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	if steps < 0 {
		err := fmt.Errorf("steps must be zero or positive, got %d", steps)
		logger.Named("migrateUpRun").Error("invalid steps", logging.Error("validateSteps", err))
		return err
	}

	// CLI オプションを反映した migrate インスタンスを組み立てます。
	m, err := newMigrator(database)
	if err != nil {
		logger.Named("migrateUpRun.buildMigrateInstance").Error("failed to create migrate instance",
			logging.Error("buildMigrateInstance", err),
		)
		return err
	}

	if steps == 0 {
		// 段数未指定なら、未適用の migration を最後まで進めます。
		logger.Named("migrateUpRun").Info("running full migration up")
	} else {
		// 段数指定時は、現在位置から指定段数分だけ Up を進めます。
		logger.Named("migrateUpRun").Info("running migration up steps", logging.Int("steps", steps))
	}
	if err := executeMigrateUp(m, steps); err != nil {
		logger.Named("migrateUpRun.executeMigrateUp").Error("migration failed",
			logging.Error("executeMigrateUp", err),
		)
		return err
	}
	logger.Named("migrateUpRun").Info("✅ migration completed")

	return nil
}

// executeMigrateUp は、steps が 0 なら全件、正なら段数指定で Up します。無変更（ErrNoChange）は成功扱いです。
func executeMigrateUp(m migrator, steps int) error {
	var err error
	if steps == 0 {
		err = m.Up()
	} else {
		err = m.Steps(steps)
	}
	// 既に最新であれば ErrNoChange になるため、両経路とも成功扱いとして握りつぶします。
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
