// Package cli は、コマンドラインのインターフェースを提供するためのパッケージです。
package cli

import (
	"boilerplate-go/internal/cli/dumpschema"
	"boilerplate-go/internal/cli/fixcollation"
	"boilerplate-go/internal/cli/mergedml"
	"boilerplate-go/internal/cli/migrate"
	"boilerplate-go/internal/cli/seed"
	"boilerplate-go/internal/cli/server"

	"github.com/spf13/cobra"
)

// RegisterCommands は、CLIのサブコマンドを登録します。
func RegisterCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(
		server.NewServeCommand(),
		migrate.NewMigrateUpCommand(),
		migrate.NewMigrateDownCommand(),
		seed.NewDBSeedCommand(),
		fixcollation.NewCommand(),
		dumpschema.NewCommand(),
		mergedml.NewCommand(),
	)
}
