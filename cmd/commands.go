package main

import "github.com/spf13/cobra"

// registerCommands は、CLIのサブコマンドを root に登録します。
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
		newOutboxRelayCommand(),
	)
}
