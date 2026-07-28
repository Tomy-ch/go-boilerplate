// Package seed は、データベース初期データ投入のコアロジックを提供します。
package seed

import (
	"context"
	"regexp"
	"sort"
	"strings"

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

// placeholderPattern は、seed ファイル中の `${NAME}` 形式のプレースホルダにマッチします。
var placeholderPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// errUndefinedPlaceholder は、seed ファイルのプレースホルダに対応する値が渡されなかった場合のエラーです。
var errUndefinedPlaceholder = xerrors.New("undefined seed placeholder")

// errUnsafePlaceholderValue は、プレースホルダの値が SQL の文字列リテラルを抜け出せる場合のエラーです。
var errUnsafePlaceholderValue = xerrors.New("unsafe seed placeholder value")

// RunDBSeed は、DB 接続の取得・seed ファイル列挙・投入のオーケストレーションを行います。
// vars は seed ファイル中の `${NAME}` に対応する値で、値の無いプレースホルダは投入せずエラーになります。
func RunDBSeed(
	logger logging.Logger,
	fsys fs.FS,
	database string,
	vars map[string]string,
	openDB func(logging.Logger, string) (driver.DatabaseDriver, error),
) error {
	ctx := context.Background()

	db, err := openDB(logger, database)
	if err != nil {
		logger.Named("dbSeedRun.dbOpen").Error(ctx, "failed to open database connection", logging.Error(logging.ErrorKey, err))
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Named("dbSeedRun.dbClose").Error(ctx, "failed to close database connection", logging.Error(logging.ErrorKey, cerr))
		}
	}()

	files, err := fsys.Glob(seedFilePlace + "/*.sql")
	if err != nil {
		logger.Named("dbSeedRun.globSeedFiles").Error(ctx, "failed to glob seed files", logging.Error(logging.ErrorKey, err))
		return err
	}

	return runSeeds(ctx, fsys, db, logger, vars, files)
}

// runSeeds は、seed ファイル群を昇順で順次実行します。実行エラーは握り潰さず呼び出し元へ返しつつ、
// 他ファイルの投入は継続します（テーブル未作成のスキップは execSeedFile 側で吸収）。
func runSeeds(
	ctx context.Context,
	fsys fs.FS,
	db driver.DatabaseDriver,
	logger logging.Logger,
	vars map[string]string,
	files []string,
) error {
	var seedErr error
	sort.Strings(files)
	for _, f := range files {
		if err := execSeedFile(ctx, fsys, db, logger, vars, f); err != nil {
			seedErr = xerrors.Join(seedErr, err)
		}
	}
	if seedErr != nil {
		return seedErr
	}
	logger.Named("dbSeedRun").Info(ctx, "✅ seeding completed")

	return nil
}

// execSeedFile は、1つの seed ファイルの読み込みとプレースホルダ展開、SQL 実行を担当します。
func execSeedFile(
	ctx context.Context,
	fsys fs.FS,
	db driver.DatabaseDriver,
	logger logging.Logger,
	vars map[string]string,
	filePath string,
) error {
	data, err := fsys.ReadFile(filePath)
	if err != nil {
		logger.Named("dbSeedRun.os.ReadFile").Error(
			ctx,
			"failed to read seed file",
			logging.String("file", filePath),
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}

	sql, err := expandPlaceholders(string(data), vars)
	if err != nil {
		logger.Named("dbSeedRun.expandPlaceholders").Error(
			ctx,
			"failed to expand seed placeholders",
			logging.String("file", filePath),
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}

	_, err = db.Exec(ctx, sql)
	return handleSeedExecResult(ctx, logger, filePath, err)
}

// expandPlaceholders は、seed ファイル中の `${NAME}` を vars の値へ置き換えます。ファイル単位で解決し、
// 1 つでも解決できないプレースホルダがあればそのファイルは実行しません。
//
// 値が無い（空文字を含む）プレースホルダは埋めずにエラーを返します。環境依存の値が空のまま投入されると、
// seed は成功したのに実行時だけ失敗する状態になり、原因の特定が難しくなるためです。
// 単一引用符を含む値も同様に拒否します。展開先は SQL テキストであり、文字列リテラルを抜け出せる値は
// 後続のステートメントとして実行されうるためです。
func expandPlaceholders(sql string, vars map[string]string) (string, error) {
	undefined := make(map[string]struct{})
	unsafe := make(map[string]struct{})
	expanded := placeholderPattern.ReplaceAllStringFunc(sql, func(placeholder string) string {
		name := placeholderPattern.FindStringSubmatch(placeholder)[1]
		value := vars[name]
		switch {
		case value == "":
			undefined[name] = struct{}{}
		case strings.Contains(value, "'"):
			unsafe[name] = struct{}{}
		default:
			return value
		}

		return placeholder
	})
	if len(undefined) > 0 {
		return "", xerrors.Wrap(errUndefinedPlaceholder, sortedNames(undefined))
	}
	if len(unsafe) > 0 {
		return "", xerrors.Wrap(errUnsafePlaceholderValue, sortedNames(unsafe))
	}

	return expanded, nil
}

// sortedNames は、プレースホルダ名の集合をエラーメッセージ用の昇順の一覧へ整えます。
func sortedNames(names map[string]struct{}) string {
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	return strings.Join(sorted, ", ")
}

// handleSeedExecResult は、seed 実行結果を PostgreSQL 固有エラーの種類に応じて記録し、
// スキップ可能なケース（成功・対象テーブル未作成）では nil を、それ以外の実行エラーでは
// そのエラーを返します。
func handleSeedExecResult(ctx context.Context, logger logging.Logger, filePath string, err error) error {
	log := logger.Named("dbSeedRun.Exec")
	if err == nil {
		log.Info(
			ctx,
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
			ctx,
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
		ctx,
		message,
		logging.String("file", filePath),
		logging.Error(logging.ErrorKey, err),
	)
	return err
}
