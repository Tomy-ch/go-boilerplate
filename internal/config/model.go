package config

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

type Config struct {
	os       OperationSystemConfig
	app      ApplicationConfig
	server   ServerConfig
	metrics  MetricsConfig
	database DatabaseConfig
	security SecurityConfig
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

// OSTimeZone は、OSのタイムゾーンを返します。
func (o *OperationSystemConfig) OSTimeZone() string { return o.timezone }

// NewApplicationConfig は、アプリケーションの設定を返します。
func NewApplicationConfig(cfg *Config) *ApplicationConfig { return &cfg.app }

// AppEnv は、サーバーの環境を返します。
//
// 例: "local", "development", "staging", "production" など。
func (a *ApplicationConfig) AppEnv() string { return a.env }

// AppMode は、アプリケーションの環境を返します。
//
// この環境変数はアプリケーションがどのモードで動作しているかを示します。
// 例: "development", "production" など。
func (a *ApplicationConfig) AppMode() string { return a.mode }

// AppName は、アプリケーションの名前を返します。
func (a *ApplicationConfig) AppName() string { return a.name }

// AppShutdownTimeout は、アプリケーションのシャットダウンタイムアウトを返します。
func (a *ApplicationConfig) AppShutdownTimeout() time.Duration { return a.shutdownTimeout }

// IsAppProductionMode は、アプリケーションが本番環境モードかどうかを返します。
func (a *ApplicationConfig) IsAppProductionMode() bool {
	return a.mode == ProductionMode
}

// IsAppDevelopmentMode は、アプリケーションが開発環境モードかどうかを返します。
func (a *ApplicationConfig) IsAppDevelopmentMode() bool {
	return a.mode == DevelopmentMode
}

// NewServerConfig は、サーバーの設定を返します。
func NewServerConfig(cfg *Config) *ServerConfig { return &cfg.server }

// ServerHost は、サーバーがリッスンするホスト名を返します。
func (s *ServerConfig) ServerHost() string { return s.host }

// ServerPort は、サーバーがリッスンするポート番号を返します。
func (s *ServerConfig) ServerPort() int { return s.port }

// ServerShutdownTimeout は、サーバー停止までの規定時間を返します。
func (s *ServerConfig) ServerShutdownTimeout() time.Duration { return s.shutdownTimeout }

// NewMetricsConfig は、メトリクスの設定を返します。
func NewMetricsConfig(cfg *Config) *MetricsConfig { return &cfg.metrics }

// MetricsHost は、メトリクスサーバーがリッスンするホスト名を返します。
func (m *MetricsConfig) MetricsHost() string { return m.host }

// MetricsPort は、メトリクスサーバーがリッスンするポート番号を返します。
func (m *MetricsConfig) MetricsPort() int { return m.port }

// NewDatabaseConfig は、データベースの設定を返します。
func NewDatabaseConfig(cfg *Config) *DatabaseConfig { return &cfg.database }

// DatabaseDriver は、データベースのドライバー名を返します。
func (d *DatabaseConfig) DatabaseDriver() string { return d.driver }

// DatabaseHost は、データベースのホスト名を返します。
func (d *DatabaseConfig) DatabaseHost() string { return d.host }

// DatabasePort は、データベースのポート番号を返します。
func (d *DatabaseConfig) DatabasePort() int { return d.port }

// DatabaseUser は、データベースのユーザー名を返します。
func (d *DatabaseConfig) DatabaseUser() string { return d.user }

// DatabasePassword は、データベースのパスワードを返します。
func (d *DatabaseConfig) DatabasePassword() string { return d.password }

// DatabaseName は、データベースの名前を返します。
func (d *DatabaseConfig) DatabaseName() string { return d.name }

// DatabaseSSLMode は、データベースのSSLモードを返します。
func (d *DatabaseConfig) DatabaseSSLMode() string { return d.sslMode }

// DatabaseDSN は、データベースの接続URLを返します。
func (d *DatabaseConfig) DatabaseDSN(o *OperationSystemConfig) string {
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

// DBMaxOpenConns は、データベースの最大オープン接続数を返します。
func (c *DBConnectionConfig) DBMaxOpenConns() int { return c.maxOpenConns }

// DBMaxIdleConns は、データベースの最大アイドル接続数を返します。
func (c *DBConnectionConfig) DBMaxIdleConns() int { return c.maxIdleConns }

// DBConnMaxLifetime は、データベースの接続の最大寿命を返します。
func (c *DBConnectionConfig) DBConnMaxLifetime() time.Duration { return c.maxLifetime }

// DBConnMaxIdleTime は、データベースの接続の最大アイドル時間を返します。
func (c *DBConnectionConfig) DBConnMaxIdleTime() time.Duration { return c.maxIdleTime }

func NewSecurityConfig(cfg *Config) *SecurityConfig { return &cfg.security }

// AllowedOrigins は、CORSを許可するオリジンのリストを返します。
func (s *SecurityConfig) AllowedOrigins() []string { return s.allowedOrigins }

// CIDR は、セキュリティ設定で使用されるCIDRを返します。
func (s *SecurityConfig) CIDR() *net.IPNet { return s.cidr }
