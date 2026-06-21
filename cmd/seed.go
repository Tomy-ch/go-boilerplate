package main

import (
	"go-boilerplate/internal/cli/seed"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/envutil"
	"go-boilerplate/pkg/fs"

	"github.com/spf13/cobra"
)

// newDBSeedCommand は、データベースに初期データを投入するためのコマンドを生成します。
func newDBSeedCommand() *cobra.Command {
	var database string

	cmd := &cobra.Command{
		Use:   "db-seed",
		Short: "データベースに初期データを投入します。",
		Long: "このコマンドは、データベースに初期データを投入するためのコマンドです。\n" +
			"--database フラグを指定すると、対象のデータベース（例: local, test）を指定して投入を行います。",
		RunE: func(_ *cobra.Command, _ []string) error {
			return dbSeedRun(database)
		},
	}

	cmd.Flags().StringVar(&database, "database", "", "filter DATABASE (e.g. local)")

	return cmd
}

// dbSeedRun は、ロガーと FS・DB 接続を組み立て、seed.RunDBSeed へ委譲する薄い殻です。
func dbSeedRun(database string) error {
	logger := logging.NewJSONLogger(logging.LevelInfo, logging.LevelError)

	return seed.RunDBSeed(logger, fs.OS{}, database, openSeedDB)
}

// openSeedDB は、seed 用設定を読み込み DB 接続を確立する実依存の口です。
func openSeedDB(logger logging.Logger, database string) (driver.DatabaseDriver, error) {
	cfg, err := newConfigForSeed(logger, database)
	if err != nil {
		return nil, err
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	return driver.NewDB(dbCfg, osCfg, dbConnCfg)
}

// newConfigForSeed は seed 用の設定を読み込み、CLI オプションの DB 名上書きを反映します。
func newConfigForSeed(logger logging.Logger, database string) (*config.Config, error) {
	if err := config.Load(); err != nil {
		logger.Named("dbSeedRun.configLoad").Error("failed to load config", logging.Error(logging.ErrorKey, err))
		return nil, err
	}
	if database != "" {
		restore, oerr := envutil.Override("DB_NAME", database)
		if oerr != nil {
			logger.Named("dbSeedRun.setenv").Error("failed to override DB_NAME env", logging.Error(logging.ErrorKey, oerr))
			return nil, oerr
		}
		defer restore()
	}
	cfg, err := config.New()
	if err != nil {
		logger.Named("dbSeedRun.configNew").Error("failed to load config", logging.Error(logging.ErrorKey, err))
		return nil, err
	}
	return cfg, nil
}
