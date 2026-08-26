// Package fixcollation は、PostgreSQL の照合順序不整合を修正するコアロジックを提供します。
package fixcollation

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/exec"
	"go-boilerplate/pkg/xerrors"
)

const (
	workDir         = "/app"
	psqlCommand     = "psql"
	callerSkipCount = 1
)

// errInvalidDatabaseName は、許可されていない DB 名が指定された場合のエラーです。
var errInvalidDatabaseName = xerrors.New("invalid database name")

// errInvalidDSN は、接続 URL を解釈できなかった場合のエラーです。
var errInvalidDSN = xerrors.New("invalid dsn")

// poolDatabasePattern は、worktree スロットが借りる開発用データベース名です。
// DB 名は SQL へ文字列として埋め込むため、識別子として安全な形のみを通します。
var poolDatabasePattern = regexp.MustCompile(`^wt[1-9][0-9]{0,2}_(local|test)$`)

// RunFix は、DB 名の検証・DSN 解決・collation 修正のオーケストレーションを行います。
// loadDSN は (パスワード非含有 DSN, パスワード) を返します。
func RunFix(ctx context.Context, runner exec.Runner, logger logging.Logger, database string, loadDSN func() (string, string, error)) error {
	if err := validateDatabaseName(database); err != nil {
		return err
	}

	dbURL, password, err := loadDSN()
	if err != nil {
		return err
	}

	targetURL, err := withDatabase(dbURL, database)
	if err != nil {
		return err
	}

	return fixCollation(ctx, runner, logger, targetURL, password, database)
}

// validateDatabaseName は、許可済みの開発・テスト用 DB 名のみを受け付けます。
// template1 は、そこから複製される DB へ不整合が伝播するため対象に含めます。
func validateDatabaseName(name string) error {
	switch name {
	case "local", "test", "template1":
		return nil
	}
	if poolDatabasePattern.MatchString(name) {
		return nil
	}
	return xerrors.Wrap(errInvalidDatabaseName, name)
}

// withDatabase は、接続 URL の接続先データベースを name へ差し替えます。
// REINDEX DATABASE は接続中のデータベースしか対象にできないため、設定由来の接続先のままでは
// 指定した DB を修正できません。
func withDatabase(dsn, name string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", xerrors.Wrap(errInvalidDSN, dsn)
	}
	u.Path = "/" + name
	return u.String(), nil
}

// fixCollation は、collation mismatch 修正 SQL を psql 経由で順に実行します。
// パスワードは引数に載せず PGPASSWORD で渡します。
func fixCollation(ctx context.Context, runner exec.Runner, logger logging.Logger, dbURL, password, database string) error {
	log := logger.CallerSkip(callerSkipCount).Named("fixcollation")
	log.Info(ctx, "start collation fix", logging.String("database", database))

	sqlStatements := []string{
		fmt.Sprintf("REINDEX DATABASE %s;", database),
		fmt.Sprintf("ALTER DATABASE %s REFRESH COLLATION VERSION;", database),
	}

	env := []string{"PGPASSWORD=" + password}
	// 依存順序があるため、照合順序修正 SQL は並列ではなく順番に流します。
	for _, sql := range sqlStatements {
		args := []string{dbURL, "-v", "ON_ERROR_STOP=1", "-c", sql}
		if _, err := runner.Output(ctx, workDir, env, psqlCommand, args); err != nil {
			log.Error(ctx, "psql command failed",
				logging.String("database", database),
				logging.String("sql", sql),
				logging.Error(logging.ErrorKey, err),
			)
			return xerrors.Wrap(err, "psql command failed")
		}
	}

	log.Info(ctx, "collation fix completed successfully", logging.String("database", database))
	return nil
}
