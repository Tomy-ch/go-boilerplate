package appconfig

type Config struct {
	server      server
	environment environment
}

type server struct {
	host           string
	port           int
	allowedOrigins []string
}

type environment struct {
	serverEnv string
	appMode   string
}

// ServerHost は、サーバーがリッスンするホスト名を返します。
func (c *Config) ServerHost() string { return c.server.host }

// ServerPort は、サーバーがリッスンするポート番号を返します。
func (c *Config) ServerPort() int { return c.server.port }

// AllowedOrigins は、CORSを許可するオリジンのリストを返します。
func (c *Config) AllowedOrigins() []string { return c.server.allowedOrigins }

// ServerEnv は、サーバーの環境を返します。
//
// 例: "local", "development", "staging", "production" など。
func (c *Config) ServerEnv() string { return c.environment.serverEnv }

// AppMode は、アプリケーションの環境を返します。
//
// この環境変数はアプリケーションがどのモードで動作しているかを示します。
// 例: "development", "production" など。
func (c *Config) AppMode() string { return c.environment.appMode }
