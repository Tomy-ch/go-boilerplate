package main

import "github.com/spf13/cobra"

// registerCommands は、CLIのサブコマンドを root に登録します。
//
// 各サブコマンドは Cobra 定義と実依存（config / DB / DI / シグナル）の結線を担う薄い殻で、
// 実ロジックは internal/cli/<command> パッケージ（テスト可能なコア）へ委譲します。
func registerCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(
		newServeCommand(),
		newMigrateUpCommand(),
		newMigrateDownCommand(),
		newDBSeedCommand(),
		newFixCollationCommand(),
		newDumpSchemaCommand(),
		newMergeDMLCommand(),
		newJobCommand(),
		newWorkerCommand(),
	)
}
