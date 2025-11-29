// Package server は、サーバーを起動するためのコマンドを提供するためのパッケージです。
package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"boilerplate-go/internal/config"
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
	cfg, err := config.SetUpConfig()
	if err != nil {
		return err
	}
	appCfg := config.NewApplicationConfig(cfg)

	var metricsSrv *http.Server
	if !appCfg.IsAppProductionMode() {
		// 非本番環境では、メトリクスサーバーを起動
		metricsSrv = MetricsServer(cfg)
		go func() {
			if err = metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("metrics server error: %v", err)
			}
		}()
	}

	app := fx.New(
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
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Interrupt,
	)
	defer stop()

	stopCtx, cancel := context.WithTimeout(
		context.Background(),
		appCfg.AppShutdownTimeout(),
	)
	defer cancel()

	if err = app.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()

	if metricsSrv != nil {
		if err = metricsSrv.Close(); err != nil {
			log.Printf("metrics server close error: %v", err)
		}
	}

	return app.Stop(stopCtx)
}
