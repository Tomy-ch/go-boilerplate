package main

import (
	"fmt"

	climigrate "go-boilerplate/internal/cli/migrate"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/envutil"

	"github.com/spf13/cobra"

	"github.com/golang-migrate/migrate/v4"
	// postgres driver for golang-migrate (required for runtime registration)
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// migrateFilePlace は、マイグレーションファイルの場所を定義します。
const migrateFilePlace = "database/migrations"

// newMigrateUpCommand は、DBのマイグレーションを上げるためのコマンドを生成します。
func newMigrateUpCommand() *cobra.Command {
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
			logger, err := logging.NewProductionLogger()
			if err != nil {
				return err
			}
			return climigrate.MigrateUpRun(steps, database, logger, buildMigrateInstance)
		},
	}

	cmd.Flags().IntVar(&steps, "steps", 0, "現在位置から Up する段数（0 で全件、正の整数のみ）")
	cmd.Flags().StringVar(&database, "database", "", "対象データベース（例: local）")

	return cmd
}

// newMigrateDownCommand は、DBのマイグレーションを下げるためのコマンドを生成します。
func newMigrateDownCommand() *cobra.Command {
	var (
		steps    int
		database string
	)

	cmd := &cobra.Command{
		Use:   "migrate-down",
		Short: "database/migrations のDDLをダウングレードします（--steps / --database指定可）。",
		Long: `database/migrations ディレクトリに存在するDDLマイグレーションを適用します。

--steps を指定しない場合（0）は、適用済みのマイグレーションを全て Down します。
--steps に正の整数を指定すると、現在位置からその段数だけ Down します。
--database フラグを指定すると、対象のデータベース（例: local, test）に対して Down を行います。`,
		RunE: func(_ *cobra.Command, _ []string) error {
			logger, err := logging.NewProductionLogger()
			if err != nil {
				return err
			}
			return climigrate.MigrateDownRun(steps, database, logger, buildMigrateInstance)
		},
	}

	cmd.Flags().IntVar(&steps, "steps", 0, "現在位置から Down する段数（0 で全件、正の整数のみ）")
	cmd.Flags().StringVar(&database, "database", "", "対象データベース（例: local）")

	return cmd
}

// buildMigrateInstance は、設定を読み込み golang-migrate のインスタンスを生成します。
func buildMigrateInstance(database string) (climigrate.Migrator, error) {
	if err := config.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if database != "" {
		// config.New() が読み取る間だけ DB_NAME を差し替え、読み取り後は元値へ復元して冪等性を保つ。
		restore, err := envutil.Override("DB_NAME", database)
		if err != nil {
			return nil, err
		}
		defer restore()
	}
	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	m, err := migrate.New("file://"+migrateFilePlace, driver.DSNWithTimeZoneString(dbCfg, osCfg))
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	return m, nil
}
