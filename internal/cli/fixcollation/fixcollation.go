// Package fixcollation は、PostgreSQL の照合順序不整合を修正するコマンドを提供します。
package fixcollation

import (
	"context"
	"fmt"

	"go-boilerplate/internal/cli/cliexec"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"

	"github.com/spf13/cobra"
)

const (
	workDir         = "/app"
	psqlCommand     = "psql"
	callerSkipCount = 1
)

// NewCommand は fix-collation コマンドを生成します。
func NewCommand() *cobra.Command {
	var database string

	cmd := &cobra.Command{
		Use:   "fix-collation",
		Short: "PostgreSQL の照合順序 (collation) バージョン不一致を修正します",
		Long: "PostgreSQL の照合順序バージョンが OS のライブラリと異なる場合に発生する mismatch を修正します。\n" +
			"具体的には REINDEX DATABASE と ALTER DATABASE ... REFRESH COLLATION VERSION を実行します。",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFixCollation(cmd.Context(), database)
		},
	}

	cmd.Flags().StringVar(&database, "database", "local", "対象データベース名（例: local）")

	return cmd
}

// runFixCollation は、設定と DSN を組み立て、collation 修正を fixCollation へ委譲する薄い殻です。
func runFixCollation(ctx context.Context, database string) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	// 想定外の DB への実行を避けるため、許可済みのローカル向け DB 名だけを受け付けます。
	if err := validateDatabaseName(database); err != nil {
		return err
	}

	cfg, err := config.SetUpConfig()
	if err != nil {
		logger.CallerSkip(callerSkipCount).Error("failed to load config", logging.Error("config", err))
		return err
	}
	dbURL := driver.DSNString(config.NewDatabaseConfig(cfg))

	return fixCollation(ctx, cliexec.OS{}, logger, dbURL, database)
}

// validateDatabaseName は、許可済みのローカル向け DB 名のみを受け付けます。
func validateDatabaseName(name string) error {
	if name == "" || name != "local" && name != "test" {
		return fmt.Errorf("invalid database name: %s", name)
	}
	return nil
}

// fixCollation は、collation mismatch 修正 SQL を psql 経由で順に実行します。
func fixCollation(ctx context.Context, runner cliexec.Runner, logger logging.Logger, dbURL, database string) error {
	logger.CallerSkip(callerSkipCount).Named("fixcollation").Info("start collation fix",
		logging.String("database", database),
	)

	sqlStatements := []string{
		fmt.Sprintf("REINDEX DATABASE %s;", database),
		fmt.Sprintf("ALTER DATABASE %s REFRESH COLLATION VERSION;", database),
	}

	// 依存順序があるため、照合順序修正 SQL は並列ではなく順番に流します。
	for _, sql := range sqlStatements {
		args := []string{dbURL, "-v", "ON_ERROR_STOP=1", "-c", sql}
		if _, err := runner.Output(ctx, workDir, psqlCommand, args); err != nil {
			logger.CallerSkip(callerSkipCount).Named("fixcollation").Error("psql command failed",
				logging.String("database", database),
				logging.String("sql", sql),
				logging.Error("psql", err),
			)
			return fmt.Errorf("psql command failed: %w", err)
		}
	}

	logger.CallerSkip(callerSkipCount).Named("fixcollation").Info("collation fix completed successfully",
		logging.String("database", database),
	)
	return nil
}
