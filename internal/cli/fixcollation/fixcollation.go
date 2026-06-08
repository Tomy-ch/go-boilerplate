// Package fixcollation は、PostgreSQL の照合順序不整合を修正するコアロジックを提供します。
package fixcollation

import (
	"context"
	"fmt"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/exec"
)

const (
	workDir         = "/app"
	psqlCommand     = "psql"
	callerSkipCount = 1
)

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

	return fixCollation(ctx, runner, logger, dbURL, password, database)
}

// validateDatabaseName は、許可済みのローカル向け DB 名のみを受け付けます。
func validateDatabaseName(name string) error {
	switch name {
	case "local", "test":
		return nil
	default:
		return fmt.Errorf("invalid database name: %s", name)
	}
}

// fixCollation は、collation mismatch 修正 SQL を psql 経由で順に実行します。
// パスワードは引数に載せず PGPASSWORD で渡します。
func fixCollation(ctx context.Context, runner exec.Runner, logger logging.Logger, dbURL, password, database string) error {
	log := logger.CallerSkip(callerSkipCount).Named("fixcollation")
	log.Info("start collation fix", logging.String("database", database))

	sqlStatements := []string{
		fmt.Sprintf("REINDEX DATABASE %s;", database),
		fmt.Sprintf("ALTER DATABASE %s REFRESH COLLATION VERSION;", database),
	}

	env := []string{"PGPASSWORD=" + password}
	// 依存順序があるため、照合順序修正 SQL は並列ではなく順番に流します。
	for _, sql := range sqlStatements {
		args := []string{dbURL, "-v", "ON_ERROR_STOP=1", "-c", sql}
		if _, err := runner.Output(ctx, workDir, env, psqlCommand, args); err != nil {
			log.Error("psql command failed",
				logging.String("database", database),
				logging.String("sql", sql),
				logging.Error("psql", err),
			)
			return fmt.Errorf("psql command failed: %w", err)
		}
	}

	log.Info("collation fix completed successfully", logging.String("database", database))
	return nil
}
