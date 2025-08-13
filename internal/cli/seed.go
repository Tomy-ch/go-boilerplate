package cli

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/middleware/logging"
	"boilerplate-go/pkg/xerror"

	"github.com/jackc/pgconn"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewDBSeedCommand は、データベースに初期データを投入するためのコマンドを生成します。
func NewDBSeedCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "db-seed",
		Short: "データベースに初期データを投入します。",
		Long: "このコマンドは、データベースに初期データを投入するためのコマンドです。\n" +
			"保存先のテーブルが存在しない場合は、保存がスキップされます。",
		RunE: dbSeedRun,
	}
}

// dbSeedRun は、データベースに初期データを投入するための実行関数です。
func dbSeedRun(_ *cobra.Command, _ []string) error {
	logger := logging.NewProductionLogger()
	xerrors := xerror.New()

	cfg, err := config.SetUpConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	db, err := sql.Open("postgres", cfg.DatabaseDSN())
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
