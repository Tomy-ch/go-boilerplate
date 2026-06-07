// Package seed は、データベースの初期データ投入に関するコマンドを提供するためのパッケージです。
package seed

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5/pgconn"

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

	// seed 実行時の設定を組み立て、必要であれば投入先 DB 名を上書きします。
	cfg, err := newConfigForSeed(logger)
	if err != nil {
		return err
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
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

	// CLI の処理では親 context が渡ってこないため、ここで seed 実行用の context を生成します。
	ctx := context.Background()

	var readFilesErr error
	// seed ファイル名の昇順で固定し、投入順序を安定させます。
	sort.Strings(files)
	for _, f := range files {
		err = execSeedFile(ctx, db, logger, f)
		if err != nil {
			// 読み込み・実行いずれの失敗も呼び出し元へ返しつつ、他ファイルの投入は継続します。
			readFilesErr = err
		}
	}
	if readFilesErr != nil {
		return readFilesErr
	}
	logger.Named("dbSeedRun").Info("✅ seeding completed")

	return nil
}

// execSeedFile は、1つの seed ファイルの読み込みと SQL 実行を担当します。
func execSeedFile(ctx context.Context, db driver.DatabaseDriver, logger logging.Logger, filePath string) error {
	//nolint:gosec // safe: seeds folder only contains project-owned SQL files
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Named("dbSeedRun.os.ReadFile").Error(
			"failed to read seed file",
			logging.String("file", filePath),
			logging.Error("os.ReadFile", err),
		)
		return err
	}

	_, err = db.Exec(ctx, string(data))
	// 実行結果を種別ごとに記録し、スキップ可能（テーブル未作成）かどうかを判定して返します。
	// 本物の実行エラーは握り潰さず呼び出し元へ伝播させ、コマンドの exit code に反映させます。
	return handleSeedExecResult(logger, filePath, err)
}

// handleSeedExecResult は、seed 実行結果を PostgreSQL 固有エラーの種類に応じて記録し、
// スキップ可能なケース（成功・対象テーブル未作成）では nil を、それ以外の実行エラーでは
// そのエラーを返します。
func handleSeedExecResult(logger logging.Logger, filePath string, err error) error {
	log := logger.Named("dbSeedRun.Exec")
	if err == nil {
		log.Info(
			"seed file executed successfully",
			logging.String("file", filePath),
		)
		return nil
	}

	var pgErr *pgconn.PgError
	isPgErr := xerrors.As(err, &pgErr)
	// seed 対象テーブルが未作成の環境では警告に留め、他の seed 実行を継続します（スキップ扱い）。
	if isPgErr && pgErr.Code == relationDoesNotExistCode {
		log.Warn(
			"table does not exist, skipping seed",
			logging.String("file", filePath),
			logging.Error("db.Exec", err),
		)
		return nil
	}

	message := "failed to exec seed file"
	if !isPgErr {
		message = "failed to exec seed file (non-postgres error)"
	}

	log.Error(
		message,
		logging.String("file", filePath),
		logging.Error("db.Exec", err),
	)
	return err
}

// newConfigForSeed は seed 用の設定を読み込み、CLI オプションの DB 名上書きを反映します。
func newConfigForSeed(logger logging.Logger) (*config.Config, error) {
	err := config.Load()
	if err != nil {
		logger.Named("dbSeedRun.configLoad").Error("failed to load config", logging.Error("configLoad", err))
		return nil, err
	}
	if targetDBintoSeed != "" {
		err = os.Setenv("DB_NAME", targetDBintoSeed)
		if err != nil {
			logger.Named("dbSeedRun.setenv").Error("failed to set DB_NAME env", logging.Error("setenv", err))
			return nil, err
		}
	}
	cfg, err := config.New()
	if err != nil {
		logger.Named("dbSeedRun.configNew").Error("failed to load config", logging.Error("configNew", err))
		return nil, err
	}
	return cfg, nil
}
