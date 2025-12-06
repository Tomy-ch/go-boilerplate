// Package seed は、データベースの初期データ投入に関するコマンドを提供するためのパッケージです。
package seed

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"
	"boilerplate-go/pkg/xerrors"

	"github.com/jackc/pgconn"

	"github.com/spf13/cobra"
)

const (
	// seedFilePlace は、シードファイルの場所を定義します。
	seedFilePlace = "database/seed"

	// PostgreSQLのエラーコード: 指定のオブジェクトが存在しない場合のコード
	relationDoesNotExistCode = "42P01"
)

var targetDBintoSeed string

// NewDBSeedCommand は、データベースに初期データを投入するためのコマンドを生成します。
func NewDBSeedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db-seed",
		Short: "データベースに初期データを投入します。",
		Long: "このコマンドは、データベースに初期データを投入するためのコマンドです。\n" +
			"--database フラグを指定すると、対象のデータベース（例: local, test）を指定して投入を行います。",
		RunE: dbSeedRun,
	}

	cmd.Flags().StringVar(&targetDBintoSeed, "database", "", "filter DATABASE (e.g. local)")

	return cmd
}

// dbSeedRun は、データベースに初期データを投入するための実行関数です。
func dbSeedRun(_ *cobra.Command, _ []string) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	err = config.Load()
	if err != nil {
		logger.Named("dbSeedRun.configLoad").Error("failed to load config", logging.Error("configLoad", err))
		return err
	}
	if targetDBintoSeed != "" {
		err = os.Setenv("DB_NAME", targetDBintoSeed)
		if err != nil {
			logger.Named("dbSeedRun.setenv").Error("failed to set DB_NAME env", logging.Error("setenv", err))
			return err
		}
	}
	cfg, err := config.New()
	if err != nil {
		logger.Named("dbSeedRun.configNew").Error("failed to load config", logging.Error("configNew", err))
		return err
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOSConfig(cfg)

	db, err := sql.Open("postgres", dbCfg.DSN(osCfg))
	if err != nil {
		logger.Named("dbSeedRun.dbOpen").Error("failed to open database connection", logging.Error("dbOpen", err))
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Named("dbSeedRun.dbClose").Error("failed to close database connection", logging.Error("dbClose", cerr))
		}
	}()

	files, err := filepath.Glob(seedFilePlace + "/*.sql")
	if err != nil {
		logger.Named("dbSeedRun.globSeedFiles").Error("failed to glob seed files", logging.Error("globSeedFiles", err))
		return err
	}

	ctx := context.Background()

	var readFilesErr error
	sort.Strings(files)
	for _, f := range files {
		//nolint:gosec // safe: seeds folder only contains project‑owned SQL files
		data, err := os.ReadFile(f)
		if err != nil {
			logger.Named("dbSeedRun.os.ReadFile").Error(
				"failed to read seed file",
				logging.String("file", f),
				logging.Error("os.ReadFile", err),
			)
			readFilesErr = err
			continue
		}
		_, err = db.ExecContext(ctx, string(data))
		log := logger.Named("dbSeedRun.ExecContext")
		if err != nil {
			var pgErr *pgconn.PgError
			if xerrors.As(err, &pgErr) &&
				pgErr.Code != relationDoesNotExistCode {
				log.Error(
					"failed to exec seed file",
					logging.String("file", f),
					logging.Error("db.ExecContext", err),
				)
			}
			log.Warn(
				"table does not exist, skipping seed",
				logging.String("file", f),
				logging.Error("db.ExecContext", err),
			)
		} else {
			log.Info(
				"seed file executed successfully",
				logging.String("file", f),
			)
		}
	}
	if readFilesErr != nil {
		return readFilesErr
	}
	logger.Named("dbSeedRun").Info("✅ seeding completed")

	return nil
}
