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
	"go.uber.org/zap"
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
	logger := logging.NewProductionLogger()

	err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	if targetDBintoSeed != "" {
		err = os.Setenv("DB_NAME", targetDBintoSeed)
		if err != nil {
			logger.Fatal("failed to set DB_NAME env", zap.Error(err))
		}
	}
	cfg, err := config.New()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOSConfig(cfg)

	db, err := sql.Open("postgres", dbCfg.DatabaseDSN(osCfg))
	if err != nil {
		logger.Panic("failed to open database connection", zap.Error(err))
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Error("failed to close database connection", zap.Error(cerr))
		}
	}()

	files, err := filepath.Glob(seedFilePlace + "/*.sql")
	if err != nil {
		logger.Panic("failed to glob seed files", zap.Error(err))
	}

	ctx := context.Background()

	sort.Strings(files)
	for _, f := range files {
		//nolint:gosec // safe: seeds folder only contains project‑owned SQL files
		data, err := os.ReadFile(f)
		if err != nil {
			logger.Panic(
				"failed to read seed file",
				zap.String("file", f),
				zap.Error(err),
			)
		}
		_, err = db.ExecContext(ctx, string(data))
		if err != nil {
			var pgErr *pgconn.PgError
			if xerrors.As(err, &pgErr) &&
				pgErr.Code != relationDoesNotExistCode {
				logger.Panic(
					"failed to exec seed file",
					zap.String("file", f),
					zap.Error(err),
				)
			}
			logger.Warn(
				"table does not exist, skipping seed",
				zap.String("file", f),
				zap.Error(err),
			)
		} else {
			logger.Info(
				"seed file executed successfully",
				zap.String("file", f),
			)
		}
	}
	logger.Info("✅ seeding completed")

	return nil
}
