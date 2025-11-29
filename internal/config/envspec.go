package config

import "time"

type Loader struct {
	OS       OperationSystem `envPrefix:"OS_"`
	App      Application     `envPrefix:"APP_"`
	Server   Server          `envPrefix:"SERVER_"`
	Metrics  Metrics         `envPrefix:"METRICS_"`
	Database Database        `envPrefix:"DB_"`
	Security Security        `envPrefix:"SECURITY_"`
}

type OperationSystem struct {
	Timezone string `env:"TZ" default:"Asia/Tokyo"`
}

type Application struct {
	Env             string        `env:"ENV,required"`
	Name            string        `env:"NAME,required"`
	Mode            string        `env:"MODE,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required"`
}

type Server struct {
	Host            string        `env:"HOST,required"`
	Port            int           `env:"PORT,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required"`
}

type Metrics struct {
	Host string `env:"HOST,required"`
	Port int    `env:"PORT,required"`
}

type Database struct {
	Driver     string     `env:"DRIVER,required"`
	Host       string     `env:"HOST,required"`
	Port       int        `env:"PORT,required"`
	User       string     `env:"USER,required"`
	Password   string     `env:"PASSWORD,required"`
	Name       string     `env:"NAME,required"`
	SSLMode    string     `env:"SSL_MODE,required"`
	Connection Connection `                        envPrefix:"CONN_"`
}

type Connection struct {
	MaxOpenConns int           `env:"MAX_OPEN,required"`
	MaxIdleConns int           `env:"MAX_IDLE,required"`
	MaxLifetime  time.Duration `env:"MAX_LIFETIME,required"`
	MaxIdleTime  time.Duration `env:"MAX_IDLE_TIME,required"`
}

type Security struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS,required" envSeparator:","`
	CIDR           string   `env:"CIDR,required"`
}
