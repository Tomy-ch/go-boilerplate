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
	security      SecurityConfig
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
	host            string
	port            int
	shutdownTimeout time.Duration
}

type MetricsConfig struct {
	host string
	port int
}

type ObservabilityConfig struct {
	enabled           bool
	targetStatusCodes []int
}

type DatabaseConfig struct {
	driver     string
	host       string
	port       int
	user       string
	password   string
	name       string
	sslMode    string
	connection DBConnectionConfig
}

type DBConnectionConfig struct {
	maxOpenConns int
	maxIdleConns int
	maxLifetime  time.Duration
	maxIdleTime  time.Duration
}

type SecurityConfig struct {
	allowedOrigins []string
	cidr           *net.IPNet
}

// NewOSConfig は、OSの設定を返します。
func NewOSConfig(cfg *Config) *OperationSystemConfig { return &cfg.os }

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

// ShutdownTimeout は、サーバー停止までの規定時間を返します。
func (s *ServerConfig) ShutdownTimeout() time.Duration { return s.shutdownTimeout }

// NewMetricsConfig は、メトリクスの設定を返します。
func NewMetricsConfig(cfg *Config) *MetricsConfig { return &cfg.metrics }

// Host は、メトリクスサーバーがリッスンするホスト名を返します。
func (m *MetricsConfig) Host() string { return m.host }

// Port は、メトリクスサーバーがリッスンするポート番号を返します。
func (m *MetricsConfig) Port() int { return m.port }

// NewObservabilityConfig は、可観測の設定を返します。
func NewObservabilityConfig(cfg *Config) *ObservabilityConfig { return &cfg.observability }

// Enabled は、可観測モードが有効かどうかを返します。
func (o *ObservabilityConfig) Enabled() bool { return o.enabled }

// TargetStatusCodes は、可観測モードで監視対象となるHTTPステータスコードのリストを返します。
func (o *ObservabilityConfig) TargetStatusCodes() []int { return o.targetStatusCodes }

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

// DSN は、データベースの接続URLを返します。
func (d *DatabaseConfig) DSN(o *OperationSystemConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&timezone=%s",
		d.user,
		d.password,
		d.host,
		d.port,
		d.name,
		d.sslMode,
		url.QueryEscape(o.timezone),
	)
}

// NewDBConnectionConfig は、データベース接続の設定を返します。
func NewDBConnectionConfig(cfg *Config) *DBConnectionConfig { return &cfg.database.connection }

// MaxOpenConns は、データベースの最大オープン接続数を返します。
func (c *DBConnectionConfig) MaxOpenConns() int { return c.maxOpenConns }

// MaxIdleConns は、データベースの最大アイドル接続数を返します。
func (c *DBConnectionConfig) MaxIdleConns() int { return c.maxIdleConns }

// MaxLifetime は、データベースの接続の最大寿命を返します。
func (c *DBConnectionConfig) MaxLifetime() time.Duration { return c.maxLifetime }

// MaxIdleTime は、データベースの接続の最大アイドル時間を返します。
func (c *DBConnectionConfig) MaxIdleTime() time.Duration { return c.maxIdleTime }

func NewSecurityConfig(cfg *Config) *SecurityConfig { return &cfg.security }

// AllowedOrigins は、CORSを許可するオリジンのリストを返します。
func (s *SecurityConfig) AllowedOrigins() []string { return s.allowedOrigins }

// CIDR は、セキュリティ設定で使用されるCIDRを返します。
func (s *SecurityConfig) CIDR() *net.IPNet { return s.cidr }
