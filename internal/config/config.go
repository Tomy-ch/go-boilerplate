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
			host:              cfg.Server.Host,
			port:              cfg.Server.Port,
			readHeaderTimeout: cfg.Server.ReadHeaderTimeout,
			readTimeout:       cfg.Server.ReadTimeout,
			writeTimeout:      cfg.Server.WriteTimeout,
			idleTimeout:       cfg.Server.IdleTimeout,
		},
		metrics: MetricsConfig{
			host: cfg.Metrics.Host,
			port: cfg.Metrics.Port,
		},
		observability: ObservabilityConfig{
			enabled:           cfg.Observability.Enabled,
			targetStatusCodes: cfg.Observability.TargetStatusCodes,
		},
		database: DatabaseConfig{
			driver:                 cfg.Database.Driver,
			host:                   cfg.Database.Host,
			port:                   cfg.Database.Port,
			user:                   cfg.Database.User,
			password:               cfg.Database.Password,
			name:                   cfg.Database.Name,
			sslMode:                cfg.Database.SSLMode,
			slowQueryWarnThreshold: cfg.Database.SlowQueryWarnThreshold,
		},
		dbconnection: DBConnectionConfig{
			maxOpenConns: cfg.DBConnection.MaxOpenConns,
			maxIdleConns: cfg.DBConnection.MaxIdleConns,
			maxLifetime:  cfg.DBConnection.MaxLifetime,
			maxIdleTime:  cfg.DBConnection.MaxIdleTime,
		},
		security: SecurityConfig{
			allowedOrigins:        cfg.Security.AllowedOrigins,
			cidr:                  v.cidr,
			contentTypeNosniff:    cfg.Security.ContentTypeNosniff,
			xFrameOptions:         cfg.Security.XFrameOptions,
			hstsMaxAge:            cfg.Security.HSTSMaxAge,
			hstsExcludeSubdomains: cfg.Security.HSTSExcludeSubdomains,
			hstsPreloadEnabled:    cfg.Security.HSTSPreloadEnabled,
			referrerPolicy:        cfg.Security.ReferrerPolicy,
		},
	}, nil
}

// validateConfig は、ConfigLoaderの内容を検証します。
func validateConfig(cfg Loader) (*validatedConfig, error) {
	if err := validateApplicationConfig(cfg.App); err != nil {
		return nil, err
	}

	if err := validateServerConfig(cfg.Server); err != nil {
		return nil, err
	}

	if err := validateDatabaseConfig(cfg.Database); err != nil {
		return nil, err
	}

	cidr, err := validateSecurityConfig(cfg.Security)
	if err != nil {
		return nil, err
	}

	return &validatedConfig{
		cidr: cidr,
	}, nil
}

// validateApplicationConfig は、アプリケーション設定を検証します。
func validateApplicationConfig(appCfg Application) error {
	if appCfg.Mode != DevelopmentMode && appCfg.Mode != ProductionMode {
		return ErrInvalidAppMode
	}
	return nil
}

// validateServerConfig は、サーバー設定を検証します。
func validateServerConfig(srvCfg Server) error {
	if srvCfg.Port < MinPort || MaxPort < srvCfg.Port {
		return ErrInvalidPortRange
	}

	if srvCfg.ReadHeaderTimeout <= 0 {
		return ErrInvalidReadHeaderTimeout
	}

	if srvCfg.ReadTimeout <= 0 {
		return ErrInvalidReadTimeout
	}

	if srvCfg.WriteTimeout <= 0 {
		return ErrInvalidWriteTimeout
	}

	if srvCfg.IdleTimeout <= 0 {
		return ErrInvalidIdleTimeout
	}

	if srvCfg.ReadHeaderTimeout > srvCfg.ReadTimeout {
		return ErrReadHeaderTimeoutExceedsReadTimeout
	}
	return nil
}

// validateDatabaseConfig は、データベース設定を検証します。
func validateDatabaseConfig(dbCfg Database) error {
	if dbCfg.SlowQueryWarnThreshold < 0 {
		return ErrInvalidSlowQueryWarnThreshold
	}
	return nil
}

// validateSecurityConfig は、セキュリティ設定を検証します。
func validateSecurityConfig(secCfg Security) (*net.IPNet, error) {
	if len(secCfg.AllowedOrigins) == 0 {
		return nil, ErrEmptyAllowedOrigins
	}

	for _, origin := range secCfg.AllowedOrigins {
		if strings.HasPrefix(origin, "http://") {
			parsedURL, err := url.Parse(origin)
			if err != nil ||
				(parsedURL.Hostname() != "localhost" && parsedURL.Hostname() != "127.0.0.1") {
				return nil, ErrHTTPOnlyAllowedForLocalhost
			}
		}
	}

	_, cidr, err := net.ParseCIDR(secCfg.CIDR)
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedToParseCIDR, err)
	}

	return cidr, nil
}
