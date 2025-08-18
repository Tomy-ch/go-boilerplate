package config

import (
	"net"
	"time"
)

type Config struct {
	os       operationSystem
	server   server
	database database
	security security
}

type operationSystem struct {
	timezone string
}

type server struct {
	env             string
	appMode         string
	host            string
	port            int
	shutdownTimeout time.Duration
}

type database struct {
	driver     string
	host       string
	port       int
	user       string
	password   string
	name       string
	sslMode    string
	connection connection
}

type connection struct {
	maxOpenConns int
	maxIdleConns int
	maxLifetime  time.Duration
	maxIdleTime  time.Duration
}

type security struct {
	allowedOrigins []string
	cidr           *net.IPNet
}

// OSTimeZone は、OSのタイムゾーンを返します。
func (c *Config) OSTimeZone() string { return c.os.timezone }

// ServerHost は、サーバーがリッスンするホスト名を返します。
func (c *Config) ServerHost() string { return c.server.host }

// ServerPort は、サーバーがリッスンするポート番号を返します。
func (c *Config) ServerPort() int { return c.server.port }

// ServerShutdownTimeout は、サーバー停止までの規定時間を返します。
func (c *Config) ServerShutdownTimeout() time.Duration { return c.server.shutdownTimeout }

// ServerEnv は、サーバーの環境を返します。
//
// 例: "local", "development", "staging", "production" など。
func (c *Config) ServerEnv() string { return c.server.env }

// ServerAppMode は、アプリケーションの環境を返します。
//
// この環境変数はアプリケーションがどのモードで動作しているかを示します。
// 例: "development", "production" など。
func (c *Config) ServerAppMode() string { return c.server.appMode }

// DatabaseDriver は、データベースのドライバー名を返します。
func (c *Config) DatabaseDriver() string { return c.database.driver }

// DatabaseHost は、データベースのホスト名を返します。
func (c *Config) DatabaseHost() string { return c.database.host }

// DatabasePort は、データベースのポート番号を返します。
func (c *Config) DatabasePort() int { return c.database.port }

// DatabaseUser は、データベースのユーザー名を返します。
func (c *Config) DatabaseUser() string { return c.database.user }

// DatabasePassword は、データベースのパスワードを返します。
func (c *Config) DatabasePassword() string { return c.database.password }

// DatabaseName は、データベースの名前を返します。
func (c *Config) DatabaseName() string { return c.database.name }

// DatabaseSSLMode は、データベースのSSLモードを返します。
func (c *Config) DatabaseSSLMode() string { return c.database.sslMode }

// DBMaxOpenConns は、データベースの最大オープン接続数を返します。
func (c *Config) DBMaxOpenConns() int { return c.database.connection.maxOpenConns }

// DBMaxIdleConns は、データベースの最大アイドル接続数を返します。
func (c *Config) DBMaxIdleConns() int { return c.database.connection.maxIdleConns }

// DBConnMaxLifetime は、データベースの接続の最大寿命を返します。
func (c *Config) DBConnMaxLifetime() time.Duration { return c.database.connection.maxLifetime }

// DBConnMaxIdleTime は、データベースの接続の最大アイドル時間を返します。
func (c *Config) DBConnMaxIdleTime() time.Duration { return c.database.connection.maxIdleTime }

// AllowedOrigins は、CORSを許可するオリジンのリストを返します。
func (c *Config) AllowedOrigins() []string {
	return c.security.allowedOrigins
}

// CIDR は、セキュリティ設定で使用されるCIDRを返します。
func (c *Config) CIDR() *net.IPNet {
	return c.security.cidr
}
