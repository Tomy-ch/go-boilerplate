package config

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

type Config struct {
	os            OperationSystemConfig
	app           ApplicationConfig
	server        ServerConfig
	metrics       MetricsConfig
	observability ObservabilityConfig
	database      DatabaseConfig
	dbconnection  DBConnectionConfig
	security      SecurityConfig
	secureCookie  SecureCookieConfig
	auth          AuthConfig
	ipRateLimit   IPRateLimitConfig
}

type OperationSystemConfig struct {
	timezone string
}

type ApplicationConfig struct {
	env             string
	name            string
	mode            string
	shutdownTimeout time.Duration
}

type ServerConfig struct {
	host              string
	port              int
	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
}

type MetricsConfig struct {
	host     string
	port     int
	userName string
	password string
}

type ObservabilityConfig struct {
	enabled             bool
	targetStatusCodes   []int
	targetStatusCodeSet map[int]bool
}

type DatabaseConfig struct {
	driver                 string
	host                   string
	port                   int
	user                   string
	password               string
	name                   string
	sslMode                string
	defaultTimeout         time.Duration
	slowQueryWarnThreshold time.Duration
}

type DBConnectionConfig struct {
	maxOpenConns int
	maxIdleConns int
	maxLifetime  time.Duration
	maxIdleTime  time.Duration
}

type SecurityConfig struct {
	allowedOrigins        []string
	cidr                  *net.IPNet
	contentTypeNosniff    string
	xFrameOptions         string
	hstsMaxAge            time.Duration
	hstsExcludeSubdomains bool
	hstsPreloadEnabled    bool
	referrerPolicy        string
}

type SecureCookieConfig struct {
	secure   *bool
	sameSite string
	domain   string
}

type AuthConfig struct {
	cookieName          string
	headerName          string
	allowedHeaderBearer bool
}

type IPRateLimitConfig struct {
	enabled         bool
	requests        int
	per             time.Duration
	burst           int
	ttl             time.Duration
	cleanupInterval time.Duration
}

// NewOperationSystemConfig は、OSの設定を返します。
func NewOperationSystemConfig(cfg *Config) *OperationSystemConfig { return &cfg.os }

// TimeZone は、OSのタイムゾーンを返します。
func (o *OperationSystemConfig) TimeZone() string { return o.timezone }

// NewApplicationConfig は、アプリケーションの設定を返します。
func NewApplicationConfig(cfg *Config) *ApplicationConfig { return &cfg.app }

// Env は、サーバーの環境を返します。
//
// 例: "local", "development", "staging", "production" など。
func (a *ApplicationConfig) Env() string { return a.env }

// Mode は、アプリケーションの環境を返します。
//
// この環境変数はアプリケーションがどのモードで動作しているかを示します。
// 例: "development", "production" など。
func (a *ApplicationConfig) Mode() string { return a.mode }

// Name は、アプリケーションの名前を返します。
func (a *ApplicationConfig) Name() string { return a.name }

// ShutdownTimeout は、アプリケーションのシャットダウンタイムアウトを返します。
func (a *ApplicationConfig) ShutdownTimeout() time.Duration { return a.shutdownTimeout }

// IsProductionMode は、アプリケーションが本番環境モードかどうかを返します。
func (a *ApplicationConfig) IsProductionMode() bool {
	return a.mode == ProductionMode
}

// IsDevelopmentMode は、アプリケーションが開発環境モードかどうかを返します。
func (a *ApplicationConfig) IsDevelopmentMode() bool {
	return a.mode == DevelopmentMode
}

// NewServerConfig は、サーバーの設定を返します。
func NewServerConfig(cfg *Config) *ServerConfig { return &cfg.server }

// Host は、サーバーがリッスンするホスト名を返します。
func (s *ServerConfig) Host() string { return s.host }

// Port は、サーバーがリッスンするポート番号を返します。
func (s *ServerConfig) Port() int { return s.port }

// ReadHeaderTimeout は、サーバーのヘッダー読み取りタイムアウトを返します。
func (s *ServerConfig) ReadHeaderTimeout() time.Duration { return s.readHeaderTimeout }

// ReadTimeout は、サーバーの読み取りタイムアウトを返します。
func (s *ServerConfig) ReadTimeout() time.Duration { return s.readTimeout }

// WriteTimeout は、サーバーの書き込みタイムアウトを返します。
func (s *ServerConfig) WriteTimeout() time.Duration { return s.writeTimeout }

// IdleTimeout は、サーバーのアイドルタイムアウトを返します。
func (s *ServerConfig) IdleTimeout() time.Duration { return s.idleTimeout }

// NewMetricsConfig は、メトリクスの設定を返します。
func NewMetricsConfig(cfg *Config) *MetricsConfig { return &cfg.metrics }

// Host は、メトリクスサーバーがリッスンするホスト名を返します。
func (m *MetricsConfig) Host() string { return m.host }

// Port は、メトリクスサーバーがリッスンするポート番号を返します。
func (m *MetricsConfig) Port() int { return m.port }

// UserName は、メトリクスサーバーの認証に使用するユーザー名を返します。
func (m *MetricsConfig) UserName() string { return m.userName }

// Password は、メトリクスサーバーの認証に使用するパスワードを返します。
func (m *MetricsConfig) Password() string { return m.password }

// NewObservabilityConfig は、可観測の設定を返します。
func NewObservabilityConfig(cfg *Config) *ObservabilityConfig { return &cfg.observability }

// Enabled は、可観測モードが有効かどうかを返します。
func (o *ObservabilityConfig) Enabled() bool { return o.enabled }

// TargetStatusCodes は、可観測モードで監視対象となるHTTPステータスコードのリストを返します。
func (o *ObservabilityConfig) TargetStatusCodes() []int { return o.targetStatusCodes }

// TargetStatusCodeSet は、可観測モードで監視対象となるHTTPステータスコードのセットを返します。
func (o *ObservabilityConfig) TargetStatusCodeSet() map[int]bool { return o.targetStatusCodeSet }

// NewDatabaseConfig は、データベースの設定を返します。
func NewDatabaseConfig(cfg *Config) *DatabaseConfig { return &cfg.database }

// Driver は、データベースのドライバー名を返します。
func (d *DatabaseConfig) Driver() string { return d.driver }

// Host は、データベースのホスト名を返します。
func (d *DatabaseConfig) Host() string { return d.host }

// Port は、データベースのポート番号を返します。
func (d *DatabaseConfig) Port() int { return d.port }

// User は、データベースのユーザー名を返します。
func (d *DatabaseConfig) User() string { return d.user }

// Password は、データベースのパスワードを返します。
func (d *DatabaseConfig) Password() string { return d.password }

// DBName は、データベースの名前を返します。
func (d *DatabaseConfig) DBName() string { return d.name }

// SSLMode は、データベースのSSLモードを返します。
func (d *DatabaseConfig) SSLMode() string { return d.sslMode }

// DefaultTimeout は、データベースのデフォルトタイムアウトを返します。
func (d *DatabaseConfig) DefaultTimeout() time.Duration { return d.defaultTimeout }

// SlowQueryWarnThreshold は、スロークエリ警告の閾値を返します。
//
// この値より長く実行されたクエリは警告レベルでログ出力されます。
// 0以下の値の場合、スロークエリ警告は無効になります。
func (d *DatabaseConfig) SlowQueryWarnThreshold() time.Duration { return d.slowQueryWarnThreshold }

// DSN は、データベースの接続URLを返します。
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.user,
		d.password,
		d.host,
		d.port,
		d.name,
		d.sslMode,
	)
}

// DSNWithTimeZone は、データベースの接続URLを返します。
func (d *DatabaseConfig) DSNWithTimeZone(o *OperationSystemConfig) string {
	return fmt.Sprintf(
		"%s&timezone=%s",
		d.DSN(),
		url.QueryEscape(o.timezone),
	)
}

// NewDBConnectionConfig は、データベース接続の設定を返します。
func NewDBConnectionConfig(cfg *Config) *DBConnectionConfig { return &cfg.dbconnection }

// MaxOpenConns は、データベースの最大オープン接続数を返します。
func (c *DBConnectionConfig) MaxOpenConns() int { return c.maxOpenConns }

// MaxIdleConns は、データベースの最大アイドル接続数を返します。
func (c *DBConnectionConfig) MaxIdleConns() int { return c.maxIdleConns }

// MaxLifetime は、データベースの接続の最大寿命を返します。
func (c *DBConnectionConfig) MaxLifetime() time.Duration { return c.maxLifetime }

// MaxIdleTime は、データベースの接続の最大アイドル時間を返します。
func (c *DBConnectionConfig) MaxIdleTime() time.Duration { return c.maxIdleTime }

// NewSecurityConfig は、セキュリティの設定を返します。
func NewSecurityConfig(cfg *Config) *SecurityConfig { return &cfg.security }

// AllowedOrigins は、CORSを許可するオリジンのリストを返します。
func (s *SecurityConfig) AllowedOrigins() []string { return s.allowedOrigins }

// CIDR は、セキュリティ設定で使用されるCIDRを返します。
func (s *SecurityConfig) CIDR() *net.IPNet { return s.cidr }

// ContentTypeNosniff は、X-Content-Type-Optionsヘッダーの値を返します。
func (s *SecurityConfig) ContentTypeNosniff() string { return s.contentTypeNosniff }

// XFrameOptions は、X-Frame-Optionsヘッダーの値を返します。
func (s *SecurityConfig) XFrameOptions() string { return s.xFrameOptions }

// HSTSMaxAge は、HSTSの最大年齢を返します。
func (s *SecurityConfig) HSTSMaxAge() time.Duration { return s.hstsMaxAge }

// HSTSExcludeSubdomains は、HSTSでサブドメインを除外するかどうかを返します。
func (s *SecurityConfig) HSTSExcludeSubdomains() bool { return s.hstsExcludeSubdomains }

// HSTSPreloadEnabled は、HSTSのプリロードが有効かどうかを返します。
func (s *SecurityConfig) HSTSPreloadEnabled() bool { return s.hstsPreloadEnabled }

// ReferrerPolicy は、Referrer-Policyヘッダーの値を返します。
func (s *SecurityConfig) ReferrerPolicy() string { return s.referrerPolicy }

// NewSecureCookieConfig は、セキュアクッキーの設定を返します。
func NewSecureCookieConfig(cfg *Config) *SecureCookieConfig { return &cfg.secureCookie }

// Secure は、Secure属性の強制設定を返します。
func (s *SecureCookieConfig) Secure() *bool { return s.secure }

// SameSite は、SameSite属性の強制設定を返します。
func (s *SecureCookieConfig) SameSite() string { return s.sameSite }

// Domain は、Domain属性の強制設定を返します。
func (s *SecureCookieConfig) Domain() string { return s.domain }

// NewAuthConfig は、認証の設定を返します。
func NewAuthConfig(cfg *Config) *AuthConfig { return &cfg.auth }

// CookieName は、認証に使用するCookie名を返します。
func (a *AuthConfig) CookieName() string { return a.cookieName }

// HeaderName は、認証に使用するヘッダー名を返します。
func (a *AuthConfig) HeaderName() string { return a.headerName }

// AllowedHeaderBearer は、認証に使用するヘッダーのBearerトークンの許可設定を返します。
func (a *AuthConfig) AllowedHeaderBearer() bool { return a.allowedHeaderBearer }

// NewIPRateLimitConfig は、IPレートリミットの設定を返します。
func NewIPRateLimitConfig(cfg *Config) *IPRateLimitConfig { return &cfg.ipRateLimit }

// Enabled は、IPレートリミットが有効かどうかを返します。
func (i *IPRateLimitConfig) Enabled() bool { return i.enabled }

// Requests は、IPレートリミットのリクエスト数を返します。
func (i *IPRateLimitConfig) Requests() int { return i.requests }

// Per は、IPレートリミットの期間を返します。
func (i *IPRateLimitConfig) Per() time.Duration { return i.per }

// Burst は、IPレートリミットのバースト数を返します。
func (i *IPRateLimitConfig) Burst() int { return i.burst }

// TTL は、IPレートリミットのエントリの有効期限を返します。
func (i *IPRateLimitConfig) TTL() time.Duration { return i.ttl }

// CleanupInterval は、IPレートリミットのエントリのクリーンアップ間隔を返します。
func (i *IPRateLimitConfig) CleanupInterval() time.Duration { return i.cleanupInterval }

// Limit は、IPレートリミットの制限値を返します。
func (i *IPRateLimitConfig) Limit() float64 {
	return float64(i.requests) / i.per.Seconds()
}
