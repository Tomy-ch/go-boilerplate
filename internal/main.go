package main

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/env"

	"go.uber.org/zap"
)

func main() {
	logger := zap.NewExample()

	err := env.Load()
	if err != nil {
		logger.Fatal("called main", zap.Error(err))
	}

	cfg, err := config.New()
	if err != nil {
		logger.Fatal("called main", zap.Error(err))
	}

	logger.Info("config", zap.String("server.host", cfg.Server.Host), zap.Int("server.port", cfg.Server.Port), zap.Strings("server.allowedOrigins", cfg.Server.AllowedOrigins))
}
