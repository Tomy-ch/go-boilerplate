package main

import (
	"context"

	"go-boilerplate/internal/cli/mergedml"
	"go-boilerplate/internal/logging"

	"github.com/spf13/cobra"
)

// newMergeDMLCommand は、merge-dml コマンドを生成します。
func newMergeDMLCommand() *cobra.Command {
	var (
		targetType string
		workDir    string
	)

	cmd := &cobra.Command{
		Use:   "merge-dml",
		Short: "DMLディレクトリ(database/dml/<repository/query_service/command_service>)のsqlファイルを対象にして、<type>ごとにマージします。",
		Long: "指定されたタイプ(repository|query_service|command_service)のDMLディレクトリ内の全サブディレクトリを走査し、\n" +
			"各カテゴリごとにSQLファイルを連結して1つのSQLファイルにまとめます。\n" +
			"生成されるファイルは database/gen/ 配下に <category>_<type>.gen.sql という名前で保存されます。",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mergeDMLRun(cmd.Context(), targetType, workDir)
		},
	}

	cmd.Flags().StringVar(&targetType, "type", "", "filter TYPE (repository|query_service|command_service)")
	_ = cmd.MarkFlagRequired("type")
	cmd.Flags().StringVar(&workDir, "work-dir", "/app", "working directory path")

	return cmd
}

// mergeDMLRun は mergedml.RunMerge への薄い委譲殻です。
func mergeDMLRun(ctx context.Context, targetType, workDir string) error {
	logger := logging.NewJSONLogger(logging.LevelInfo(), logging.LevelError())

	gen := mergedml.NewGenerator(logger, workDir)
	return mergedml.RunMerge(ctx, gen, targetType)
}
