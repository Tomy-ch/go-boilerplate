// Package serve は、サーバーを起動するためのコマンドを提供するためのパッケージです。
package serve

import (
	"boilerplate-go/internal/di"

	"github.com/spf13/cobra"

	"go.uber.org/fx"
)

// NewServeCommand は、サーバーを起動するためのコマンドを生成します。
func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "サーバーを起動します。",
		Long:  "このコマンドは、アプリケーションのサーバーを起動します。",
		RunE:  serveRun,
	}
}

// serveRun は、サーバーを起動するための実行関数です。
func serveRun(_ *cobra.Command, _ []string) error {
	fx.New(
		// Core Module
		di.ConfigModule(),
		di.DatabaseModule(),
		di.LoggingModule(),
		di.HTTPStackModule(),
		// DDD Modules
		di.RepositoryModule(),
		di.UsecaseModule(),
		di.ControllerModule(),
		// Server Module
		di.ServeModule(),
	).Run()
	return nil
}
