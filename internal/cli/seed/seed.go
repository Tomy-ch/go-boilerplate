// Package seed は、データベース初期データ投入のコアロジックを提供します。
package seed

import (
	"context"
	"errors"
	"sort"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/fs"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	seedFilePlace = "database/seed"

	// PostgreSQLのエラーコード: 指定のオブジェクトが存在しない場合のコード
	relationDoesNotExistCode = "42P01"
)

// RunDBSeed は、DB 接続の取得・seed ファイル列挙・投入のオーケストレーションを行います。
func RunDBSeed(
	logger logging.Logger,
	fsys fs.FS,
	database string,
	openDB func(logging.Logger, string) (driver.DatabaseDriver, error),
) error {
	db, err := openDB(logger, database)
	if err != nil {
		logger.Named("dbSeedRun.dbOpen").Error("failed to open database connection", logging.Error(logging.ErrorKey, err))
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Named("dbSeedRun.dbClose").Error("failed to close database connection", logging.Error(logging.ErrorKey, cerr))
		}
	}()

	files, err := fsys.Glob(seedFilePlace + "/*.sql")
	if err != nil {
		logger.Named("dbSeedRun.globSeedFiles").Error("failed to glob seed files", logging.Error(logging.ErrorKey, err))
		return err
	}

	ctx := context.Background()
	return runSeeds(ctx, fsys, db, logger, files)
}

// runSeeds は、seed ファイル群を昇順で順次実行します。実行エラーは握り潰さず呼び出し元へ返しつつ、
// 他ファイルの投入は継続します（テーブル未作成のスキップは execSeedFile 側で吸収）。
func runSeeds(ctx context.Context, fsys fs.FS, db driver.DatabaseDriver, logger logging.Logger, files []string) error {
	var seedErr error
	sort.Strings(files)
	for _, f := range files {
		if err := execSeedFile(ctx, fsys, db, logger, f); err != nil {
			seedErr = errors.Join(seedErr, err)
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
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}

	_, err = db.Exec(ctx, string(data))
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
			logging.Error(logging.ErrorKey, err),
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
		logging.Error(logging.ErrorKey, err),
	)
	return err
}
