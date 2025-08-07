// Package appconfig は、アプリケーションの設定を管理します。
package appconfig

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

func New() (*Config, error) {
	cfg, err := env.ParseAs[ConfigLoader]()
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedToParseConfig, err)
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
		database: database{
			host:     cfg.Database.Host,
			port:     cfg.Database.Port,
			user:     cfg.Database.User,
			password: cfg.Database.Password,
			name:     cfg.Database.Name,
			sslMode:  cfg.Database.SSLMode,
		},
	}, nil
}

func validateConfig(cfg ConfigLoader) error {
	if cfg.Server.Port < MinPort || cfg.Server.Port > MaxPort {
		return ErrInvalidPortRange
	}

	for _, origin := range cfg.Server.AllowedOrigins {
		if strings.HasPrefix(origin, "http://") {
			parsedURL, err := url.Parse(origin)
			if err != nil ||
				(parsedURL.Hostname() != "localhost" && parsedURL.Hostname() != "127.0.0.1") {

				return ErrHTTPOnlyAllowedForLocalhost
			}
		}
	}

	if cfg.Environment.AppMode != DevelopmentMode &&
		cfg.Environment.AppMode != ProductionMode {

		return ErrInvalidAppMode
	}

	return nil
}

func (c *Config) IsAppProductionMode() bool {
	return c.environment.appMode == ProductionMode
}

func (c *Config) IsAppDevelopmentMode() bool {
	return c.environment.appMode == DevelopmentMode
}
