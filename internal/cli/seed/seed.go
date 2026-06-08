// Package seed は、データベースの初期データ投入に関するコマンドを提供するためのパッケージです。
package seed

import (
	"context"
	"os"
	"sort"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/fs"
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

// NewDBSeedCommand は、データベースに初期データを投入するためのコマンドを生成します。
func NewDBSeedCommand() *cobra.Command {
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

// dbSeedRun は、ロガーと実依存（FS・DB 接続）を組み立て、seed 投入を runDBSeed へ委譲する薄い殻です。
func dbSeedRun(database string) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	return runDBSeed(logger, fs.OS{}, database, openSeedDB)
}

// openSeedDB は、seed 用設定を読み込み DB 接続を確立する実依存の口です。
func openSeedDB(logger logging.Logger, database string) (driver.DatabaseDriver, error) {
	// seed 実行時の設定を組み立て、必要であれば投入先 DB 名を上書きします。
	cfg, err := newConfigForSeed(logger, database)
	if err != nil {
		return nil, err
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	return driver.NewDB(dbCfg, osCfg, dbConnCfg)
}

// runDBSeed は、DB 接続の取得・seed ファイル列挙・投入のオーケストレーションを行います。
func runDBSeed(
	logger logging.Logger,
	fsys fs.FS,
	database string,
	openDB func(logging.Logger, string) (driver.DatabaseDriver, error),
) error {
	db, err := openDB(logger, database)
	if err != nil {
		logger.Named("dbSeedRun.dbOpen").Error("failed to open database connection", logging.Error("dbOpen", err))
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Named("dbSeedRun.dbClose").Error("failed to close database connection", logging.Error("dbClose", cerr))
		}
	}()

	files, err := fsys.Glob(seedFilePlace + "/*.sql")
	if err != nil {
		logger.Named("dbSeedRun.globSeedFiles").Error("failed to glob seed files", logging.Error("globSeedFiles", err))
		return err
	}

	// CLI の処理では親 context が渡ってこないため、ここで seed 実行用の context を生成します。
	ctx := context.Background()
	return runSeeds(ctx, fsys, db, logger, files)
}

// runSeeds は、seed ファイル群を昇順で順次実行します。実行エラーは握り潰さず呼び出し元へ返しつつ、
// 他ファイルの投入は継続します（テーブル未作成のスキップは execSeedFile 側で吸収）。
func runSeeds(ctx context.Context, fsys fs.FS, db driver.DatabaseDriver, logger logging.Logger, files []string) error {
	var seedErr error
	// seed ファイル名の昇順で固定し、投入順序を安定させます。
	sort.Strings(files)
	for _, f := range files {
		if err := execSeedFile(ctx, fsys, db, logger, f); err != nil {
			seedErr = err
		}
	}
	if seedErr != nil {
		return seedErr
	}
	logger.Named("dbSeedRun").Info("✅ seeding completed")

	return nil
}

// execSeedFile は、1つの seed ファイルの読み込みと SQL 実行を担当します。
func execSeedFile(ctx context.Context, fsys fs.FS, db driver.DatabaseDriver, logger logging.Logger, filePath string) error {
	data, err := fsys.ReadFile(filePath)
	if err != nil {
		logger.Named("dbSeedRun.os.ReadFile").Error(
			"failed to read seed file",
			logging.String("file", filePath),
			logging.Error("os.ReadFile", err),
		)
		return err
	}

	_, err = db.Exec(ctx, string(data))
	// 本物の実行エラーは握り潰さず呼び出し元へ伝播させ、テーブル未作成のみスキップ扱いにします。
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
func newConfigForSeed(logger logging.Logger, database string) (*config.Config, error) {
	err := config.Load()
	if err != nil {
		logger.Named("dbSeedRun.configLoad").Error("failed to load config", logging.Error("configLoad", err))
		return nil, err
	}
	if database != "" {
		err = os.Setenv("DB_NAME", database)
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
