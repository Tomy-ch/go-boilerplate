// Package config は、アプリケーションの設定を管理します。
package config

import (
	"net"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
	"golang.org/x/crypto/bcrypt"

	"go-boilerplate/pkg/xerrors"
)

// exporterNone は、送出を明示的に無効化する exporter 値。
const exporterNone = "none"

// New は、アプリケーションの設定を初期化します。
func New() (*Config, error) {
	cfg, err := env.ParseAs[Loader]()
	if err != nil {
		return nil, xerrors.Join(ErrFailedToParseConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// CIDR は解析が検証を兼ねる（parse, don't validate）。値が必要な New で
	// 一度だけ解析し、検証済みの *net.IPNet を直接 Config へ格納する。
	cidr, err := parseCIDR(cfg.Security.CIDR)
	if err != nil {
		return nil, err
	}

	return &Config{
		os: OperatingSystemConfig{
			timezone: cfg.OS.Timezone,
		},
		app: ApplicationConfig{
			env:             cfg.App.Env,
			name:            cfg.App.Name,
			mode:            cfg.App.Mode,
			logLevel:        cfg.App.LogLevel,
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
			host:     cfg.Metrics.Host,
			port:     cfg.Metrics.Port,
			userName: cfg.Metrics.UserName,
			password: cfg.Metrics.Password,
		},
		observability: ObservabilityConfig{
			tracesExporter:      cfg.Observability.TracesExporter,
			metricsExporter:     cfg.Observability.MetricsExporter,
			logsExporter:        cfg.Observability.LogsExporter,
			otlpEndpoint:        cfg.Observability.OTLPEndpoint,
			otlpProtocol:        cfg.Observability.OTLPProtocol,
			maskedDBQueryArgs:   cfg.Observability.MaskedDBQueryArgs,
			targetStatusCodeSet: buildStatusCodeSet(cfg.Observability.TargetStatusCodes),
		},
		database: DatabaseConfig{
			driver:                 cfg.Database.Driver,
			host:                   cfg.Database.Host,
			port:                   cfg.Database.Port,
			user:                   cfg.Database.User,
			password:               cfg.Database.Password,
			name:                   cfg.Database.Name,
			sslMode:                cfg.Database.SSLMode,
			pingTimeout:            cfg.Database.PingTimeout,
			slowQueryWarnThreshold: cfg.Database.SlowQueryWarnThreshold,
		},
		dbconnection: DBConnectionConfig{
			maxConns:    cfg.DBConnection.MaxConns,
			minConns:    cfg.DBConnection.MinConns,
			maxLifetime: cfg.DBConnection.MaxLifetime,
			maxIdleTime: cfg.DBConnection.MaxIdleTime,
		},
		security: SecurityConfig{
			allowedOrigins:        cfg.Security.AllowedOrigins,
			cidr:                  cidr,
			contentTypeNosniff:    cfg.Security.ContentTypeNosniff,
			xFrameOptions:         cfg.Security.XFrameOptions,
			hstsMaxAge:            cfg.Security.HSTSMaxAge,
			hstsExcludeSubdomains: cfg.Security.HSTSExcludeSubdomains,
			hstsPreloadEnabled:    cfg.Security.HSTSPreloadEnabled,
			referrerPolicy:        cfg.Security.ReferrerPolicy,
			bcryptCost:            cfg.Security.BcryptCost,
		},
		secureCookie: SecureCookieConfig{
			secure:   cfg.SecureCookie.Secure,
			sameSite: cfg.SecureCookie.SameSite,
			domain:   cfg.SecureCookie.Domain,
		},
		auth: AuthConfig{
			cookieName:          cfg.Auth.CookieName,
			headerName:          cfg.Auth.HeaderName,
			allowedHeaderBearer: cfg.Auth.AllowedHeaderBearer,
		},
		worker: WorkerConfig{
			concurrency:               cfg.Worker.Concurrency,
			maxInFlight:               cfg.Worker.MaxInFlight,
			batchSize:                 cfg.Worker.BatchSize,
			extendInterval:            cfg.Worker.ExtendInterval,
			drainTimeout:              cfg.Worker.DrainTimeout,
			receiveCountWarnThreshold: cfg.Worker.ReceiveCountWarnThreshold,
			circuitFailureThreshold:   cfg.Worker.CircuitFailureThreshold,
			circuitOpenBackoffInitial: cfg.Worker.CircuitOpenBackoffInitial,
			circuitOpenBackoffMax:     cfg.Worker.CircuitOpenBackoffMax,
			circuitHalfOpenProbe:      cfg.Worker.CircuitHalfOpenProbe,
			healthListenAddr:          cfg.Worker.HealthListenAddr,
			progressStaleAfter:        cfg.Worker.ProgressStaleAfter,
			nackBackoffInitial:        cfg.Worker.NackBackoffInitial,
			nackBackoffMax:            cfg.Worker.NackBackoffMax,
		},
		outbox: OutboxConfig{
			endpoint:     cfg.Outbox.Endpoint,
			pollInterval: cfg.Outbox.PollInterval,
			errorBackoff: cfg.Outbox.ErrorBackoff,
			batchSize:    cfg.Outbox.BatchSize,
		},
	}, nil
}

// validateConfig は、Loaderの内容を検証します。
func validateConfig(cfg Loader) error {
	if err := validateApplicationConfig(cfg.App); err != nil {
		return err
	}

	if err := validateServerConfig(cfg.Server); err != nil {
		return err
	}

	if err := validateDatabaseConfig(cfg.Database); err != nil {
		return err
	}

	if err := validateDBConnectionConfig(cfg.DBConnection); err != nil {
		return err
	}

	if err := validateSecurityConfig(cfg.Security); err != nil {
		return err
	}

	if err := validateAuthConfig(cfg.Auth); err != nil {
		return err
	}

	return nil
}

// validateApplicationConfig は、アプリケーション設定を検証します。
func validateApplicationConfig(appCfg Application) error {
	if appCfg.Mode != DevelopmentMode && appCfg.Mode != ProductionMode {
		return ErrInvalidAppMode
	}
	switch appCfg.LogLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel
	}
	return nil
}

// validatePortRange は、ポート番号が許容範囲内かを検証します。
func validatePortRange(port int, outOfRangeErr error) error {
	if port < MinPort || MaxPort < port {
		return outOfRangeErr
	}
	return nil
}

// validateServerConfig は、サーバー設定を検証します。
func validateServerConfig(srvCfg Server) error {
	if err := validatePortRange(srvCfg.Port, ErrInvalidPortRange); err != nil {
		return err
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
	if err := validatePortRange(dbCfg.Port, ErrInvalidDBPortRange); err != nil {
		return err
	}
	if dbCfg.PingTimeout <= 0 {
		return ErrInvalidDBPingTimeout
	}
	if dbCfg.SlowQueryWarnThreshold < 0 {
		return ErrInvalidSlowQueryWarnThreshold
	}
	return nil
}

// validateDBConnectionConfig は、データベース接続設定を検証します。
func validateDBConnectionConfig(dbConnCfg DBConnection) error {
	if dbConnCfg.MinConns > dbConnCfg.MaxConns {
		return ErrInvalidExceedMaxConns
	}
	return nil
}

// validateSecurityConfig は、セキュリティ設定を検証します。
// CIDR は解析が検証を兼ねるため、ここでは扱わず New の parseCIDR に委ねる。
func validateSecurityConfig(secCfg Security) error {
	if len(secCfg.AllowedOrigins) == 0 {
		return ErrEmptyAllowedOrigins
	}

	if secCfg.BcryptCost < bcrypt.MinCost || bcrypt.MaxCost < secCfg.BcryptCost {
		return ErrInvalidBcryptCost
	}

	for _, origin := range secCfg.AllowedOrigins {
		parsedURL, err := url.Parse(origin)
		if err != nil {
			return ErrHTTPOnlyAllowedForLocalhost
		}
		// スキームは url.Parse で小文字正規化されるが、念のため EqualFold で大小無視判定する。
		if strings.EqualFold(parsedURL.Scheme, "http") &&
			parsedURL.Hostname() != "localhost" && parsedURL.Hostname() != "127.0.0.1" {
			return ErrHTTPOnlyAllowedForLocalhost
		}
	}

	return nil
}

// parseCIDR は、CIDR 文字列を *net.IPNet へ解析します。
func parseCIDR(s string) (*net.IPNet, error) {
	_, cidr, err := net.ParseCIDR(s)
	if err != nil {
		return nil, xerrors.Join(ErrFailedToParseCIDR, err)
	}
	return cidr, nil
}

// validateAuthConfig は、認証設定を検証します。
func validateAuthConfig(authCfg Auth) error {
	if authCfg.CookieName == "" && authCfg.HeaderName == "" {
		return ErrAuthConfigMissing
	}
	return nil
}

// buildStatusCodeSet は、HTTPステータスコードのセットを構築します。
func buildStatusCodeSet(codes []int) map[int]bool {
	m := make(map[int]bool, len(codes))
	for _, code := range codes {
		m[code] = true
	}
	return m
}

// isActiveExporter は、exporter 指定が送出を行う有効値かどうかを返します。
// 空文字（未設定）と "none" は無効（送出しない）とみなします。
func isActiveExporter(v string) bool {
	return v != "" && !strings.EqualFold(v, exporterNone)
}
