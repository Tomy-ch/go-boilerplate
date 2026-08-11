// Package config は、アプリケーションの設定を管理します。
package config

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"

	"go-boilerplate/pkg/xerrors"
)

// exporterNone は、送出を明示的に無効化する exporter 値。
const exporterNone = "none"

// New は、環境変数から設定を読み込み、検証したうえで Config を構築して返します。
// 型変換の失敗、値の範囲・相互整合性の違反、CIDR の解析失敗のいずれでもエラーを返します。
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
			bodyLimitMB:       cfg.Server.BodyLimitMB,
			requestTimeout:    cfg.Server.RequestTimeout,
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
			statementTimeout:       cfg.Database.StatementTimeout,
			lockTimeout:            cfg.Database.LockTimeout,
			txMaxRetries:           cfg.Database.TxMaxRetries,
			txRetryBaseBackoff:     cfg.Database.TxRetryBaseBackoff,
			txRetryMaxBackoff:      cfg.Database.TxRetryMaxBackoff,
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
		},
		secureCookie: SecureCookieConfig{
			secure:   cfg.SecureCookie.Secure,
			sameSite: cfg.SecureCookie.SameSite,
			domain:   cfg.SecureCookie.Domain,
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
		consumerQueue: ConsumerQueueConfig{
			endpoint:          cfg.ConsumerQueue.Endpoint,
			region:            cfg.ConsumerQueue.Region,
			url:               cfg.ConsumerQueue.URL,
			dlqURL:            cfg.ConsumerQueue.DLQURL,
			accessKeyID:       cfg.ConsumerQueue.AccessKeyID,
			secretAccessKey:   cfg.ConsumerQueue.SecretAccessKey,
			maxMessages:       cfg.ConsumerQueue.MaxMessages,
			waitTimeSeconds:   cfg.ConsumerQueue.WaitTimeSeconds,
			visibilityTimeout: cfg.ConsumerQueue.VisibilityTimeout,
		},
		outbox: OutboxConfig{
			publisher:            cfg.Outbox.Publisher,
			endpoint:             cfg.Outbox.Endpoint,
			pollInterval:         cfg.Outbox.PollInterval,
			errorBackoff:         cfg.Outbox.ErrorBackoff,
			batchSize:            cfg.Outbox.BatchSize,
			queueEndpoint:        cfg.Outbox.QueueEndpoint,
			queueRegion:          cfg.Outbox.QueueRegion,
			queueURL:             cfg.Outbox.QueueURL,
			queueAccessKeyID:     cfg.Outbox.QueueAccessKeyID,
			queueSecretAccessKey: cfg.Outbox.QueueSecretAccessKey,
		},
		auth: AuthConfig{
			issuer:             cfg.Auth.Issuer,
			audience:           cfg.Auth.Audience,
			jwksURL:            cfg.Auth.JWKSURL,
			allowedAlgorithms:  cfg.Auth.AllowedAlgorithms,
			clockSkew:          cfg.Auth.ClockSkew,
			jwksCacheTTL:       cfg.Auth.JWKSCacheTTL,
			discoveryTTL:       cfg.Auth.JWKSDiscoveryTTL,
			unknownKidCooldown: cfg.Auth.JWKSUnknownKIDCooldown,
		},
		objectStorage: ObjectStorageConfig{
			endpoint:        cfg.ObjectStorage.Endpoint,
			region:          cfg.ObjectStorage.Region,
			bucket:          cfg.ObjectStorage.Bucket,
			accessKeyID:     cfg.ObjectStorage.AccessKeyID,
			secretAccessKey: cfg.ObjectStorage.SecretAccessKey,
			usePathStyle:    cfg.ObjectStorage.UsePathStyle,
			maxUploadBytes:  cfg.ObjectStorage.MaxUploadBytes,
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

	if err := validateMetricsConfig(cfg.Metrics); err != nil {
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

	if err := validateEmbeddedEnv(cfg.App); err != nil {
		return err
	}

	return nil
}

// validateEmbeddedEnv は、production モードでバイナリに焼き込まれた env の素性を検証します。
// 実効モードが production かつ埋め込み env の APP_ENV が非本番（deny-list）の場合、
// materialize-env 忘れによりローカル値が本番へ紛れ込んでいるとみなしエラーを返します。
// deny 型のため、未知の新環境ラベルには寛容です。development モードは実行時注入を優先する設計思想により全て許容します。
func validateEmbeddedEnv(appCfg Application) error {
	if appCfg.Mode != ProductionMode {
		return nil
	}

	switch embeddedAppEnv {
	case EnvLocal, EnvCI, EnvTest, EnvDast, EnvDevelopment, "":
		return ErrEmbeddedEnvMismatch
	default:
		return nil
	}
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

	if srvCfg.WriteTimeout < srvCfg.RequestTimeout {
		return ErrWriteTimeoutBelowRequestTimeout
	}
	return nil
}

// validateMetricsConfig は、メトリクスサーバーのポート番号が有効範囲内であることを検証します。
func validateMetricsConfig(metricsCfg Metrics) error {
	return validatePortRange(metricsCfg.Port, ErrInvalidMetricsPortRange)
}

// ValidateServerShutdown は、graceful shutdown 猶予が処理中リクエストの予算を下回らないことを検証します。
// 値の妥当性ルールは config の責務だが、この制約は HTTP サーバーを組むプロセスにのみ意味を持つため、
// New() では全プロファイル共通に走らせない。
func ValidateServerShutdown(appCfg *ApplicationConfig, srvCfg *ServerConfig) error {
	return validateServerShutdown(appCfg.ShutdownTimeout(), srvCfg.RequestTimeout())
}

// validateServerShutdown は、shutdown >= request を検証します。
func validateServerShutdown(shutdown, request time.Duration) error {
	if shutdown < request {
		return ErrShutdownTimeoutBelowRequestTimeout
	}
	return nil
}

// ValidateUploadBodyLimit は、リクエストボディ上限がアップロード上限を上回ることを検証します。
// ValidateServerShutdown と同じ理由で、New() では走らせない。
func ValidateUploadBodyLimit(srvCfg *ServerConfig, objCfg *ObjectStorageConfig) error {
	return validateUploadBodyLimit(srvCfg.BodyLimitMB(), objCfg.MaxUploadBytes())
}

// validateUploadBodyLimit は、bodyLimitMB をバイト換算した値が maxUploadBytes を上回ることを検証します。
// マルチパートのオーバーヘッド分をどれだけ積むかは運用判断のため、ここでは等値も不可とするだけに留めます。
func validateUploadBodyLimit(bodyLimitMB int, maxUploadBytes int64) error {
	if int64(bodyLimitMB)*BytesPerMB <= maxUploadBytes {
		return ErrBodyLimitBelowMaxUploadBytes
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
