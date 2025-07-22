package main

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler/health"
	"boilerplate-go/internal/controller/handler/healthz"
	"boilerplate-go/internal/controller/middleware/logging"
	"boilerplate-go/internal/controller/server"
	"boilerplate-go/internal/env"

	"go.uber.org/zap"
)

func main() {
	l := zap.NewExample()

	err := env.Load()
	if err != nil {
		l.Fatal("called main", zap.Error(err))
	}

	cfg, err := config.New()
	if err != nil {
		l.Fatal("called main", zap.Error(err))
	}

	logger, err := logging.NewDevelopmentLogger()
	if err != nil {
		l.Fatal("called main", zap.Error(err))
	}

	e := server.New(cfg, logger)

	health.BindHandler(e)
	healthz.BindHandler(e)

	if err := e.Start(":8080"); err != nil {
		l.Fatal("called main", zap.Error(err))
	}
}
