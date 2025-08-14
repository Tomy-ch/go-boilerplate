// Package cli は、コマンドラインのインターフェースを提供するためのパッケージです。
package cli

import (
	"boilerplate-go/internal/cli/gensqlc"
	"boilerplate-go/internal/cli/migrate"
	"boilerplate-go/internal/cli/seed"
	"boilerplate-go/internal/cli/serve"

	"github.com/spf13/cobra"
)

// RegisterCommands は、CLIのサブコマンドを登録します。
func RegisterCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(
		serve.NewServeCommand(),
		migrate.NewMigrateUpCommand(),
		migrate.NewMigrateDownCommand(),
		seed.NewDBSeedCommand(),
		gensqlc.NewCommand(),
	)
}
