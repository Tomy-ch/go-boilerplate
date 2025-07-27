package appconfig

type ConfigLoader struct {
	Server      Server
	Environment Environment
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
