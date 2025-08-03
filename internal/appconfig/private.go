package appconfig

type ConfigLoader struct {
	Server      Server
	Environment Environment
	Database    Database `envPrefix:"DB_"`
}

type Environment struct {
	ServerEnv string `env:"ENV,required"`
	AppMode   string `env:"APP_MODE,required"`
}

type Server struct {
	Host           string   `env:"HOST,required"`
	Port           int      `env:"PORT,required"`
	AllowedOrigins []string `env:"ALLOWED_ORIGINS,required" envSeparator:","`
}

type Database struct {
	Host     string `env:"HOST,required"`
	Port     int    `env:"PORT,required"`
	User     string `env:"USER,required"`
	Password string `env:"PASSWORD,required"`
	Name     string `env:"NAME,required"`
	SSLMode  string `env:"SSLMODE,required"`
}
