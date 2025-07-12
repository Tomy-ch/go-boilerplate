package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

func New() (*Config, error) {
	cfg, err := env.ParseAs[ConfigLoader]()
	if err != nil {
		return nil, fmt.Errorf("%w : %v", ErrFailedToParseConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return &Config{
		environment: environment{
			serverEnv: cfg.Environment.ServerEnv,
			appMode:   cfg.Environment.AppMode,
		},
		server: server{
			host:           cfg.Server.Host,
			port:           cfg.Server.Port,
			allowedOrigins: cfg.Server.AllowedOrigins,
		},
	}, nil
}

func validateConfig(cfg ConfigLoader) error {
	if cfg.Server.Port < MinPort || cfg.Server.Port > MaxPort {
		return ErrInvalidPortRange
	}

	for _, origin := range cfg.Server.AllowedOrigins {
		if strings.HasPrefix(origin, "http://") && origin != "http://localhost" {
			return ErrHTTPOnlyAllowedForLocalhost
		}
	}

	if cfg.Environment.AppMode != DevelopmentMode && cfg.Environment.AppMode != ProductionMode {
		return ErrInvalidAppMode
	}

	return nil
}

func (c *Config) IsAppEnvProduction() bool {
	return c.environment.appMode == ProductionMode
}

func (c *Config) IsAppEnvDevelopment() bool {
	return c.environment.appMode == DevelopmentMode
}
