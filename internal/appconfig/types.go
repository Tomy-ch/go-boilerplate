package appconfig

import (
	"fmt"
	"net"
)

type Config struct {
	server      server
	environment environment
	database    database
	security    security
}

type server struct {
	host string
	port int
}

type environment struct {
	serverEnv string
	appMode   string
}

type database struct {
	host     string
	port     int
	user     string
	password string
	name     string
	sslMode  string
}

type security struct {
	allowedOrigins []string
	cidr           *net.IPNet
}

// ServerHost は、サーバーがリッスンするホスト名を返します。
func (c *Config) ServerHost() string { return c.server.host }

// ServerPort は、サーバーがリッスンするポート番号を返します。
func (c *Config) ServerPort() int { return c.server.port }

// ServerEnv は、サーバーの環境を返します。
//
// 例: "local", "development", "staging", "production" など。
func (c *Config) ServerEnv() string { return c.environment.serverEnv }

// AppMode は、アプリケーションの環境を返します。
//
// この環境変数はアプリケーションがどのモードで動作しているかを示します。
// 例: "development", "production" など。
func (c *Config) AppMode() string { return c.environment.appMode }

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

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DatabaseUser(),
		c.DatabasePassword(),
		c.DatabaseHost(),
		c.DatabasePort(),
		c.DatabaseName(),
		c.DatabaseSSLMode(),
	)
}

// AllowedOrigins は、CORSを許可するオリジンのリストを返します。
func (c *Config) AllowedOrigins() []string {
	return c.security.allowedOrigins
}

// CIDR は、セキュリティ設定で使用されるCIDRを返します。
func (c *Config) CIDR() *net.IPNet {
	return c.security.cidr
}
