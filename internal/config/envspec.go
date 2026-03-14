package config

import "time"

type Loader struct {
	OS            OperationSystem `envPrefix:"OS_"`
	App           Application     `envPrefix:"APP_"`
	Server        Server          `envPrefix:"SERVER_"`
	Observability Observability   `envPrefix:"OBSERVABILITY_"`
	Metrics       Metrics         `envPrefix:"METRICS_"`
	Database      Database        `envPrefix:"DB_"`
	DBConnection  DBConnection    `envPrefix:"DBCONN_"`
	Security      Security        `envPrefix:"SECURITY_"`
	SecureCookie  SecureCookie    `envPrefix:"SECURE_COOKIE_"`
	Auth          Auth            `envPrefix:"AUTH_"`
	IPRateLimit   IPRateLimit     `envPrefix:"IP_RATE_LIMITER_"`
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
	Host              string        `env:"HOST,required"`
	Port              int           `env:"PORT,required"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT,required"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT,required"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT,required"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT,required"`
}

type Metrics struct {
	Host     string `env:"HOST,required"`
	Port     int    `env:"PORT,required"`
	UserName string `env:"USERNAME,required"`
	Password string `env:"PASSWORD,required"`
}

type Observability struct {
	Enabled           bool  `env:"ENABLED,required"`
	TargetStatusCodes []int `env:"TARGET_STATUS_CODES,required" envSeparator:","`
}

type Database struct {
	Driver                 string        `env:"DRIVER,required"`
	Host                   string        `env:"HOST,required"`
	Port                   int           `env:"PORT,required"`
	User                   string        `env:"USER,required"`
	Password               string        `env:"PASSWORD,required"`
	Name                   string        `env:"NAME,required"`
	SSLMode                string        `env:"SSL_MODE,required"`
	SlowQueryWarnThreshold time.Duration `env:"SLOW_QUERY_WARN_THRESHOLD,required"`
}

type DBConnection struct {
	MaxOpenConns int           `env:"MAX_OPEN,required"`
	MaxIdleConns int           `env:"MAX_IDLE,required"`
	MaxLifetime  time.Duration `env:"MAX_LIFETIME,required"`
	MaxIdleTime  time.Duration `env:"MAX_IDLE_TIME,required"`
}

type Security struct {
	AllowedOrigins        []string      `env:"ALLOWED_ORIGINS,required"         envSeparator:","`
	CIDR                  string        `env:"CIDR,required"`
	ContentTypeNosniff    string        `env:"CONTENT_TYPE_NOSNIFF,required"`
	XFrameOptions         string        `env:"X_FRAME_OPTIONS,required"`
	HSTSMaxAge            time.Duration `env:"HSTS_MAX_AGE,required"`
	HSTSExcludeSubdomains bool          `env:"HSTS_EXCLUDE_SUBDOMAINS,required"`
	HSTSPreloadEnabled    bool          `env:"HSTS_PRELOAD_ENABLED,required"`
	ReferrerPolicy        string        `env:"REFERRER_POLICY,required"`
	BcryptCost            int           `env:"BCRYPT_COST,required"`
}

type SecureCookie struct {
	Secure   *bool  `env:"SECURE"`
	SameSite string `env:"SAME_SITE"`
	Domain   string `env:"DOMAIN"`
}

type Auth struct {
	CookieName          string `env:"COOKIE_NAME,required"`
	HeaderName          string `env:"HEADER_NAME,required"`
	AllowedHeaderBearer bool   `env:"ALLOWED_HEADER_BEARER,required"`
}

type IPRateLimit struct {
	Enabled         bool          `env:"ENABLED,required"`
	Requests        int           `env:"REQUESTS,required"`
	Per             time.Duration `env:"PER,required"`
	Burst           int           `env:"BURST,required"`
	TTL             time.Duration `env:"TTL,required"`
	CleanupInterval time.Duration `env:"CLEANUP_INTERVAL,required"`
}
