// Package serve は、サーバーを起動するためのコマンドを提供するためのパッケージです。
package serve

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler/health"
	"boilerplate-go/internal/controller/handler/healthz"
	v1users "boilerplate-go/internal/controller/handler/v1/users"
	"boilerplate-go/internal/controller/middleware"
	"boilerplate-go/internal/controller/middleware/binder"
	"boilerplate-go/internal/controller/middleware/errorhandler"
	"boilerplate-go/internal/controller/middleware/ipextractor"
	"boilerplate-go/internal/controller/middleware/logging"
	"boilerplate-go/internal/controller/middleware/validator"
	"boilerplate-go/internal/controller/server"
	"boilerplate-go/internal/infrastructure/rdb"

	"github.com/spf13/cobra"

	"go.uber.org/zap"
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
	logger := logging.NewProductionLogger()

	cfg, err := config.SetUpConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.NamedError("config", err))
	}

	validator := validator.NewValidator()
	binder := binder.NewBinder()
	ipextractor := ipextractor.NewIPExtractor(cfg)
	httpErrorHandler := errorhandler.NewHTTPErrorHandler(logger)

	e := server.New(cfg, validator, binder, ipextractor, httpErrorHandler)
	middleware.UseMiddlewares(e, cfg, logger)

	db, err := rdb.NewDB(cfg)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.NamedError("database", err))
	}

	health.BindHandler(e)
	healthz.BindHandler(e)

	v1users.BindHandler(e, db)

	if err := e.Start(":8080"); err != nil {
		logger.Fatal("called main", zap.Error(err))
	}

	return nil
}
