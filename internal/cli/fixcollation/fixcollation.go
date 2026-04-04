// Package fixcollation は、PostgreSQL の照合順序不整合を修正するコマンドを提供します。
package fixcollation

import (
	"context"
	"fmt"
	"os"
	"os/exec"

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

var targetDB string

// NewCommand は fix-collation コマンドを生成します。
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix-collation",
		Short: "PostgreSQL の照合順序 (collation) バージョン不一致を修正します",
		Long: "PostgreSQL の照合順序バージョンが OS のライブラリと異なる場合に発生する mismatch を修正します。\n" +
			"具体的には REINDEX DATABASE と ALTER DATABASE ... REFRESH COLLATION VERSION を実行します。",
		RunE: runFixCollation,
	}

	cmd.Flags().StringVar(&targetDB, "database", "local", "対象データベース名（例: local）")

	return cmd
}

// runFixCollation は、実際に collation mismatch 修正 SQL を実行します。
func runFixCollation(_ *cobra.Command, _ []string) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	cfg, err := config.SetUpConfig()
	if err != nil {
		logger.CallerSkip(callerSkipCount).Error("failed to load config", logging.Error("config", err))
		return err
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	dbURL := driver.DSNString(dbCfg)

	ctx := context.Background()

	runPSQL := func(sql string) error {
		// #nosec G204 -- dbURL from config, sql controlled by code
		cmd := exec.CommandContext(ctx, psqlCommand, dbURL, "-v", "ON_ERROR_STOP=1", "-c", sql)
		cmd.Dir = workDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	logger.CallerSkip(callerSkipCount).Named("fixcollation").Info("start collation fix",
		logging.String("database", targetDB),
	)

	sqlStatements := []string{
		fmt.Sprintf("REINDEX DATABASE %s;", targetDB),
		fmt.Sprintf("ALTER DATABASE %s REFRESH COLLATION VERSION;", targetDB),
	}

	// 照合順序修正SQLを順次実行
	for _, sql := range sqlStatements {
		if err := runPSQL(sql); err != nil {
			logger.CallerSkip(callerSkipCount).Named("fixcollation").Error("psql command failed",
				logging.String("database", targetDB),
				logging.String("sql", sql),
				logging.Error("psql", err),
			)
			return fmt.Errorf("psql command failed: %w", err)
		}
	}

	logger.CallerSkip(callerSkipCount).Named("fixcollation").Info("collation fix completed successfully",
		logging.String("database", targetDB),
	)
	return nil
}
