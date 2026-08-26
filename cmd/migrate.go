package main

import (
	root "go-boilerplate"
	climigrate "go-boilerplate/internal/cli/migrate"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/envutil"
	"go-boilerplate/pkg/xerrors"

	"github.com/spf13/cobra"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	// postgres driver for golang-migrate (required for runtime registration)
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

// migrateFilePlace は、埋め込み FS 内のマイグレーションディレクトリのパスです。
const migrateFilePlace = "database/migrations"

// newCLIConfig は、設定を読み込み、CLI オプションの DB 名上書きを反映した Config を生成します。
// database が空でない場合は環境変数 DB_NAME を一時的に上書きしてから設定を構築します。
func newCLIConfig(database string) (*config.Config, error) {
	if err := config.Load(); err != nil {
		return nil, xerrors.Wrap(err, "failed to load config")
	}
	if database != "" {
		restore, err := envutil.Override("DB_NAME", database)
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to override DB_NAME env var")
		}
		defer restore()
	}
	cfg, err := config.New()
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to build config")
	}
	return cfg, nil
}

// newMigrateUpCommand は、DBのマイグレーションを上げるためのコマンドを生成します。
func newMigrateUpCommand() *cobra.Command {
	return newMigrateCommand(
		"migrate-up",
		"database/migrations のDDLをアップグレードします（--steps / --database指定可）。",
		`database/migrations ディレクトリに存在するDDLマイグレーションを適用します。

--steps を指定しない場合（0）は、未適用のマイグレーションを全て Up します。
--steps に正の整数を指定すると、現在位置からその段数だけ Up します。
--database フラグを指定すると、対象のデータベース（例: local, test）に対して Up を行います。`,
		"現在位置から Up する段数（0 で全件、正の整数のみ）",
		climigrate.MigrateUpRun,
	)
}

// newMigrateDownCommand は、DBのマイグレーションを下げるためのコマンドを生成します。
func newMigrateDownCommand() *cobra.Command {
	return newMigrateCommand(
		"migrate-down",
		"database/migrations のDDLをダウングレードします（--steps / --database指定可）。",
		`database/migrations ディレクトリに存在するDDLマイグレーションを適用します。

--steps を指定しない場合（0）は、適用済みのマイグレーションを全て Down します。
--steps に正の整数を指定すると、現在位置からその段数だけ Down します。
--database フラグを指定すると、対象のデータベース（例: local, test）に対して Down を行います。`,
		"現在位置から Down する段数（0 で全件、正の整数のみ）",
		climigrate.MigrateDownRun,
	)
}

// newMigrateCommand は、Up/Down 共通のマイグレーションコマンドを生成します。
// run には climigrate.MigrateUpRun / MigrateDownRun のいずれかを渡します。
func newMigrateCommand(
	use, short, long, stepsHelp string,
	run func(steps int, database string, logger logging.Logger, newMigrator climigrate.MigratorFactory) error,
) *cobra.Command {
	var (
		steps    int
		database string
	)

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		RunE: func(_ *cobra.Command, _ []string) error {
			logger := logging.NewJSONLogger(logging.LevelInfo(), logging.LevelError(), nil)
			return run(steps, database, logger, buildMigrateInstance)
		},
	}

	cmd.Flags().IntVar(&steps, "steps", 0, stepsHelp)
	cmd.Flags().StringVar(&database, "database", "", "対象データベース（例: local）")

	return cmd
}

// buildMigrateInstance は、設定を読み込み golang-migrate のインスタンスを生成します。
func buildMigrateInstance(database string) (climigrate.Migrator, error) {
	cfg, err := newCLIConfig(database)
	if err != nil {
		return nil, err
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)

	src, err := iofs.New(root.FS, migrateFilePlace)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to create migration source")
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, driver.DSNWithTimeZoneString(dbCfg, osCfg))
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to create migrate instance")
	}
	return m, nil
}
