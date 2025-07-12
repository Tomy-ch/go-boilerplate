package main

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler"
	"boilerplate-go/internal/controller/middleware/logging"
	"boilerplate-go/internal/controller/server"
	"boilerplate-go/internal/env"
	errUtil "boilerplate-go/pkg/errutil"

	"go.uber.org/zap"
)

func main() {
	l := zap.NewExample()

	xerrors := errUtil.CockroachDBError{}

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

	handler.RegisterRoutes(e, xerrors)

	if err := e.Start(":8080"); err != nil {
		l.Fatal("called main", zap.Error(err))
	}
}
