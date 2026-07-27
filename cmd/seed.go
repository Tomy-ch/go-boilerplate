package main

import (
	"context"

	"go-boilerplate/internal/cli/seed"
	"go-boilerplate/internal/config"
	objectstorageinfra "go-boilerplate/internal/infrastructure/objectstorage"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
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

// dbSeedRun は seed.RunDBSeed への薄い委譲殻です。
func dbSeedRun(database string) error {
	logger := logging.NewJSONLogger(logging.LevelInfo(), logging.LevelError(), nil)

	if err := seed.RunDBSeed(logger, fs.OS{}, database, openSeedDB); err != nil {
		return err
	}

	endpoint, put, err := openSeedObjectStorage(logger, database)
	if err != nil {
		return err
	}

	return seed.RunObjectSeed(logger, fs.OS{}, endpoint, put)
}

// openSeedObjectStorage は、seed 用設定を読み込みオブジェクトストレージ実装を組み立てる実依存の口です。
// 併せて接続先エンドポイントを返し、実環境のバケットへ投入しない判断を呼び出し先へ委ねます。
func openSeedObjectStorage(logger logging.Logger, database string) (string, seed.PutObjectFunc, error) {
	cfg, err := newConfigForSeed(logger, database)
	if err != nil {
		return "", nil, err
	}
	osCfg := config.NewObjectStorageConfig(cfg)
	storage := objectstorageinfra.New(osCfg, observability.NewDisabledTracerFactory())

	put := func(ctx context.Context, obj seed.ObjectToPut) error {
		_, perr := storage.Put(ctx, objectstorage.PutObject{
			Key:         obj.Key,
			Body:        obj.Body,
			ContentType: obj.ContentType,
		})
		return perr
	}

	return osCfg.Endpoint(), put, nil
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
	// CLI 設定ロード時点では trace span は無いため context.Background() を用いる。
	ctx := context.Background()
	if err := config.Load(); err != nil {
		logger.Named("dbSeedRun.configLoad").Error(ctx, "failed to load config", logging.Error(logging.ErrorKey, err))
		return nil, err
	}
	if database != "" {
		restore, oerr := envutil.Override("DB_NAME", database)
		if oerr != nil {
			logger.Named("dbSeedRun.setenv").Error(ctx, "failed to override DB_NAME env", logging.Error(logging.ErrorKey, oerr))
			return nil, oerr
		}
		defer restore()
	}
	cfg, err := config.New()
	if err != nil {
		logger.Named("dbSeedRun.configNew").Error(ctx, "failed to load config", logging.Error(logging.ErrorKey, err))
		return nil, err
	}
	return cfg, nil
}
