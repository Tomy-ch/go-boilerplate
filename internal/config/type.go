package config

type Config struct {
	Server Server
}

type Server struct {
	Host           string   `env:"HOST,required"`
	Port           int      `env:"PORT,required"`
	AllowedOrigins []string `env:"ALLOWED_ORIGINS,required" envSeparator:","`
}
