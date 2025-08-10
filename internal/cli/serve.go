package cli

import (
	"boilerplate-go/internal/bootstrap"
	"boilerplate-go/internal/controller/handler/health"
	"boilerplate-go/internal/controller/handler/healthz"
	"boilerplate-go/internal/controller/middleware"
	"boilerplate-go/internal/controller/middleware/logging"
	"boilerplate-go/internal/controller/server"

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

	cfg, err := bootstrap.SetUpConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	validator := server.NewValidator()
	binder := server.NewBinder()
	ipextractor := server.NewIPExtractor(cfg)
	httpErrorHandler := server.NewHTTPErrorHandler(logger)

	e := server.New(cfg, validator, binder, ipextractor, httpErrorHandler)
	middleware.UseMiddlewares(e, cfg, logger)

	health.BindHandler(e)
	healthz.BindHandler(e)

	if err := e.Start(":8080"); err != nil {
		logger.Fatal("called main", zap.Error(err))
	}

	return nil
}
