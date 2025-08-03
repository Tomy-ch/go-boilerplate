// Package cli は、コマンドラインのインターフェースを提供するためのパッケージです。
package cli

import (
	"boilerplate-go/internal/bootstrap"

	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
)

const (
	// migrateFilePlace は、マイグレーションファイルの場所を定義します。
	migrateFilePlace = "database/migrations"
	// seedFilePlace は、シードファイルの場所を定義します。
	seedFilePlace = "database/seed"

	// PostgreSQLのエラーコード: 指定のオブジェクトが存在しない場合のコード
	relationDoesNotExistCode = "42P01"
)

// RegisterCommands は、CLIのサブコマンドを登録します。
func RegisterCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(
		NewServeCommand(),
		NewMigrateUpCommand(),
		NewMigrateDownCommand(),
		NewDBSeedCommand(),
	)
}

// buildMigrateInstance は、マイグレーションインスタンスを生成します。
func buildMigrateInstance() (*migrate.Migrate, error) {
	cfg, err := bootstrap.SetUpConfig()
	if err != nil {
		return nil, err
	}
	return migrate.New("file://"+migrateFilePlace, cfg.DatabaseURL())
}
