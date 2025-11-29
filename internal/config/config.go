// Package config は、アプリケーションの設定を管理します。
package config

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
	cfg, err := env.ParseAs[Loader]()
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedToParseConfig, err)
	}

	v, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Config{
		os: OperationSystemConfig{
			timezone: cfg.OS.Timezone,
		},
		app: ApplicationConfig{
			env:             cfg.App.Env,
			mode:            cfg.App.Mode,
			shutdownTimeout: cfg.App.ShutdownTimeout,
		},
		server: ServerConfig{
			host:            cfg.Server.Host,
			port:            cfg.Server.Port,
			shutdownTimeout: cfg.Server.ShutdownTimeout,
		},
		metrics: MetricsConfig{
			host: cfg.Metrics.Host,
			port: cfg.Metrics.Port,
		},
		database: DatabaseConfig{
			driver:   cfg.Database.Driver,
			host:     cfg.Database.Host,
			port:     cfg.Database.Port,
			user:     cfg.Database.User,
			password: cfg.Database.Password,
			name:     cfg.Database.Name,
			sslMode:  cfg.Database.SSLMode,
			connection: DBConnectionConfig{
				maxOpenConns: cfg.Database.Connection.MaxOpenConns,
				maxIdleConns: cfg.Database.Connection.MaxIdleConns,
				maxLifetime:  cfg.Database.Connection.MaxLifetime,
				maxIdleTime:  cfg.Database.Connection.MaxIdleTime,
			},
		},
		security: SecurityConfig{
			allowedOrigins: cfg.Security.AllowedOrigins,
			cidr:           v.cidr,
		},
	}, nil
}

// validateConfig は、ConfigLoaderの内容を検証します。
func validateConfig(cfg Loader) (*validatedConfig, error) {
	if cfg.Server.Port < MinPort || MaxPort < cfg.Server.Port {
		return nil, ErrInvalidPortRange
	}

	if len(cfg.Security.AllowedOrigins) == 0 {
		return nil, ErrEmptyAllowedOrigins
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

	if cfg.App.Mode != DevelopmentMode && cfg.App.Mode != ProductionMode {
		return nil, ErrInvalidAppMode
	}

	if cfg.App.ShutdownTimeout.Microseconds() < cfg.Server.ShutdownTimeout.Microseconds() {
		return nil, ErrServerErrShutdownTimeoutExceedsApplication
	}

	_, cidr, err := net.ParseCIDR(cfg.Security.CIDR)
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedToParseCIDR, err)
	}

	return &validatedConfig{
		cidr: cidr,
	}, nil
}
