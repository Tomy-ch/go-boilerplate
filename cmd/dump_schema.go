package main

import (
	"context"

	"go-boilerplate/internal/cli/dumpschema"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"

	"github.com/spf13/cobra"
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
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	gen := dumpschema.NewGenerator(logger, workDir)

	loadDSN := func() (string, error) {
		cfg, cerr := config.SetUpConfig()
		if cerr != nil {
			logger.Named("dumpschema.SetUpConfig").Error("failed to load config", logging.Error("config", cerr))
			return "", cerr
		}
		return driver.DSNString(config.NewDatabaseConfig(cfg)), nil
	}

	return dumpschema.RunDump(ctx, gen, loadDSN)
}
