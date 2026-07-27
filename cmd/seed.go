package main

import (
	"context"

	"go-boilerplate/internal/cli/seed"
	"go-boilerplate/internal/config"
	s3storage "go-boilerplate/internal/infrastructure/objectstorage/s3" // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"                  // sample-api:line
	"go-boilerplate/internal/usecase/boundary/objectstorage" // sample-api:line
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

	// sample-api:begin
	endpoint, put, err := openSeedObjectStorage(logger, database)
	if err != nil {
		return err
	}
	if serr := seed.RunProductImageSeed(logger, fs.OS{}, database, endpoint, put, openSeedDB); serr != nil {
		return serr
	}
	// sample-api:end

	return nil
}

// sample-api:begin
// openSeedObjectStorage は、seed 用設定を読み込みオブジェクトストレージ実装を組み立てる実依存の口です。
// 併せて接続先エンドポイントを返し、実環境のバケットへサンプル画像を投入しない判断を呼び出し先へ委ねます。
func openSeedObjectStorage(logger logging.Logger, database string) (string, seed.PutObjectFunc, error) {
	cfg, err := newConfigForSeed(logger, database)
	if err != nil {
		return "", nil, err
	}
	osCfg := config.NewObjectStorageConfig(cfg)
	storage := s3storage.New(s3storage.Config{
		Endpoint:        osCfg.Endpoint(),
		Region:          osCfg.Region(),
		Bucket:          osCfg.Bucket(),
		AccessKeyID:     osCfg.AccessKeyID(),
		SecretAccessKey: osCfg.SecretAccessKey(),
		UsePathStyle:    osCfg.UsePathStyle(),
	}, observability.NewDisabledTracerFactory())

	put := func(ctx context.Context, obj seed.ObjectToPut) error {
		_, perr := storage.Put(ctx, objectstorage.PutObject{
			Key:          obj.Key,
			Body:         obj.Body,
			ContentType:  obj.ContentType,
			CacheControl: obj.CacheControl,
		})
		return perr
	}

	return osCfg.Endpoint(), put, nil
}

// sample-api:end

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
