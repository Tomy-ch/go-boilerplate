package main

import (
	"context"

	"go-boilerplate/internal/cli/dumpschema"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

// newDumpSchemaCommand は、dump-schema コマンドを生成します。
func newDumpSchemaCommand() *cobra.Command {
	var workDir string

	cmd := &cobra.Command{
		Use:   "dump-schema",
		Short: "databaseに接続してスキーマをダンプして読み込みやすい形に整形します。",
		Long: "ファイルで定義されたdumpコマンドを実行してDBスキーマをダンプし、\n" +
			"メタコマンドの行を除去してsqlcで読み込みやすい形に整形します。\n" +
			"dumpコマンドを変更したい場合は、dumpschema パッケージの dumpCommand/dumpSubArgs を修正してください。",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDumpSchema(cmd.Context(), workDir)
		},
	}

	cmd.Flags().StringVar(&workDir, "work-dir", "/app", "working directory path")

	return cmd
}

// runDumpSchema は、ロガー・ジェネレーターと設定読込を結線し、dumpschema.RunDump へ委譲する薄い殻です。
func runDumpSchema(ctx context.Context, workDir string) error {
	logger := logging.NewJSONLogger(zapcore.InfoLevel, zapcore.ErrorLevel)

	gen := dumpschema.NewGenerator(logger, workDir)

	loadDSN := func() (string, string, error) {
		cfg, cerr := config.SetUpConfig()
		if cerr != nil {
			logger.Named("dumpschema.SetUpConfig").Error("failed to load config", logging.Error(logging.ErrorKey, cerr))
			return "", "", cerr
		}
		dbCfg := config.NewDatabaseConfig(cfg)
		return driver.DSNStringWithoutPassword(dbCfg), dbCfg.Password(), nil
	}

	return dumpschema.RunDump(ctx, gen, loadDSN)
}
