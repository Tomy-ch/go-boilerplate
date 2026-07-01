package main

import (
	"context"

	"go-boilerplate/internal/cli/fixcollation"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/exec"

	"github.com/spf13/cobra"
)

// newFixCollationCommand は fix-collation コマンドを生成します。
func newFixCollationCommand() *cobra.Command {
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

// runFixCollation は fixcollation.RunFix への薄い委譲殻です。
func runFixCollation(ctx context.Context, database string) error {
	logger := logging.NewJSONLogger(logging.LevelInfo(), logging.LevelError())

	loadDSN := func() (string, string, error) {
		cfg, cerr := config.SetUpConfig()
		if cerr != nil {
			logger.Error("failed to load config", logging.Error(logging.ErrorKey, cerr))
			return "", "", cerr
		}
		dbCfg := config.NewDatabaseConfig(cfg)
		return driver.DSNStringWithoutPassword(dbCfg), dbCfg.Password(), nil
	}

	return fixcollation.RunFix(ctx, exec.OS{}, logger, database, loadDSN)
}
