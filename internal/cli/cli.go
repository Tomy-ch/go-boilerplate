// Package cli は、コマンドラインのインターフェースを提供するためのパッケージです。
package cli

import (
	"go-boilerplate/internal/cli/dumpschema"
	"go-boilerplate/internal/cli/fixcollation"
	"go-boilerplate/internal/cli/job"
	"go-boilerplate/internal/cli/mergedml"
	"go-boilerplate/internal/cli/migrate"
	"go-boilerplate/internal/cli/seed"
	"go-boilerplate/internal/cli/server"

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
		job.NewCommand(),
	)
}
