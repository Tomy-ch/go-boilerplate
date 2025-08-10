// Package appconfig は、アプリケーションの設定を管理します。
package appconfig

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

type validatedConfig struct {
	cidr *net.IPNet
}

// New は、アプリケーションの設定を初期化します。
func New() (*Config, error) {
	cfg, err := env.ParseAs[ConfigLoader]()
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedToParseConfig, err)
	}

	v, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Config{
		environment: environment{
			serverEnv: cfg.Environment.ServerEnv,
			appMode:   cfg.Environment.AppMode,
		},
		server: server{
			host: cfg.Server.Host,
			port: cfg.Server.Port,
		},
		database: database{
			host:     cfg.Database.Host,
			port:     cfg.Database.Port,
			user:     cfg.Database.User,
			password: cfg.Database.Password,
			name:     cfg.Database.Name,
			sslMode:  cfg.Database.SSLMode,
		},
		security: security{
			allowedOrigins: cfg.Security.AllowedOrigins,
			cidr:           v.cidr,
		},
	}, nil
}

// validateConfig は、ConfigLoaderの内容を検証します。
func validateConfig(cfg ConfigLoader) (*validatedConfig, error) {
	if cfg.Server.Port < MinPort || cfg.Server.Port > MaxPort {
		return nil, ErrInvalidPortRange
	}

	for _, origin := range cfg.Security.AllowedOrigins {
		if strings.HasPrefix(origin, "http://") {
			parsedURL, err := url.Parse(origin)
			if err != nil ||
				(parsedURL.Hostname() != "localhost" && parsedURL.Hostname() != "127.0.0.1") {

				return nil, ErrHTTPOnlyAllowedForLocalhost
			}
		}
	}

	if cfg.Environment.AppMode != DevelopmentMode &&
		cfg.Environment.AppMode != ProductionMode {

		return nil, ErrInvalidAppMode
	}

	_, cidr, err := net.ParseCIDR(cfg.Security.CIDR)
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedToParseCIDR, err)
	}

	return &validatedConfig{
		cidr: cidr,
	}, nil
}

func (c *Config) IsAppProductionMode() bool {
	return c.environment.appMode == ProductionMode
}

func (c *Config) IsAppDevelopmentMode() bool {
	return c.environment.appMode == DevelopmentMode
}
