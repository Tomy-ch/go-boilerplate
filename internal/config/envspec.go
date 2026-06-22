package config

import "time"

// Loader はアプリケーション全体の設定を環境変数から読み込むルートコンテナ。
type Loader struct {
	OS            OperatingSystem `envPrefix:"OS_"`
	App           Application     `envPrefix:"APP_"`
	Server        Server          `envPrefix:"SERVER_"`
	Observability Observability   `envPrefix:"OBSERVABILITY_"`
	Metrics       Metrics         `envPrefix:"METRICS_"`
	Database      Database        `envPrefix:"DB_"`
	DBConnection  DBConnection    `envPrefix:"DBCONN_"`
	Security      Security        `envPrefix:"SECURITY_"`
	SecureCookie  SecureCookie    `envPrefix:"SECURE_COOKIE_"`
	Auth          Auth            `envPrefix:"AUTH_"`
	Worker        Worker          `envPrefix:"WORKER_"`
}

// Worker は worker engine の engine-core 設定（broker 非依存）を保持する。
// serve/job では使わないため、未設定でも起動できるよう default を与える（required にしない）。
type Worker struct {
	Concurrency               int           `env:"CONCURRENCY"                  default:"4"`
	MaxInFlight               int           `env:"MAX_IN_FLIGHT"                default:"8"`
	BatchSize                 int           `env:"BATCH_SIZE"                   default:"4"`
	ExtendInterval            time.Duration `env:"EXTEND_INTERVAL"              default:"0s"`
	DrainTimeout              time.Duration `env:"DRAIN_TIMEOUT"                default:"30s"`
	ReceiveCountWarnThreshold int           `env:"RECEIVE_COUNT_WARN_THRESHOLD" default:"5"`
	CircuitFailureThreshold   int           `env:"CIRCUIT_FAILURE_THRESHOLD"    default:"10"`
	CircuitOpenBackoffInitial time.Duration `env:"CIRCUIT_OPEN_BACKOFF_INITIAL" default:"1s"`
	CircuitOpenBackoffMax     time.Duration `env:"CIRCUIT_OPEN_BACKOFF_MAX"     default:"30s"`
	CircuitHalfOpenProbe      int           `env:"CIRCUIT_HALF_OPEN_PROBE"      default:"1"`
	HealthListenAddr          string        `env:"HEALTH_LISTEN_ADDR"           default:":8081"`
	ProgressStaleAfter        time.Duration `env:"PROGRESS_STALE_AFTER"         default:"60s"`
}

// OperatingSystem はOS レベルの設定を保持する。現時点ではタイムゾーンのみを管理する。
type OperatingSystem struct {
	Timezone string `env:"TZ" default:"Asia/Tokyo"`
}

// Application はアプリケーション識別・動作モードおよびシャットダウン制御に関する設定を保持する。
type Application struct {
	Env             string        `env:"ENV,required"`
	Name            string        `env:"NAME,required"`
	Mode            string        `env:"MODE,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required"`
}

// Server は HTTP サーバーのバインドアドレスおよび各種タイムアウトに関する設定を保持する。
type Server struct {
	Host              string        `env:"HOST,required"`
	Port              int           `env:"PORT,required"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT,required"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT,required"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT,required"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT,required"`
}

// Metrics はメトリクスエンドポイント（Prometheus 等）への接続情報と認証情報を保持する。
type Metrics struct {
	Host     string `env:"HOST,required"`
	Port     int    `env:"PORT,required"`
	UserName string `env:"USERNAME,required"`
	Password string `env:"PASSWORD,required"`
}

// Observability はトレースや計装の有効化フラグ、DBクエリ引数マスク設定、
// およびトレース対象とする HTTP ステータスコード一覧を保持する。
type Observability struct {
	Enabled           bool  `env:"ENABLED,required"`
	MaskedDBQueryArgs bool  `env:"MASKED_DB_QUERY_ARGS,required"`
	TargetStatusCodes []int `env:"TARGET_STATUS_CODES,required"  envSeparator:","`
}

// Database はデータベースへの接続先情報（ドライバ・ホスト・認証情報・SSL）および
// 接続確認タイムアウトやスロークエリ警告閾値を保持する。
type Database struct {
	Driver                 string        `env:"DRIVER,required"`
	Host                   string        `env:"HOST,required"`
	Port                   int           `env:"PORT,required"`
	User                   string        `env:"USER,required"`
	Password               string        `env:"PASSWORD,required"`
	Name                   string        `env:"NAME,required"`
	SSLMode                string        `env:"SSL_MODE,required"`
	PingTimeout            time.Duration `env:"PING_TIMEOUT,required"`
	SlowQueryWarnThreshold time.Duration `env:"SLOW_QUERY_WARN_THRESHOLD,required"`
}

// DBConnection はコネクションプールの上限・下限数とコネクションの最大生存時間・最大アイドル時間を保持する。
type DBConnection struct {
	MaxConns    int32         `env:"MAX_CONNS,required"`
	MinConns    int32         `env:"MIN_CONNS,required"`
	MaxLifetime time.Duration `env:"MAX_LIFETIME,required"`
	MaxIdleTime time.Duration `env:"MAX_IDLE_TIME,required"`
}

// Security は CORS 許可オリジン・CIDR 制限・セキュリティヘッダー（HSTS・X-Frame-Options 等）・
// Referrer Policy および bcrypt コストパラメータを保持する。
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

// SecureCookie はセキュアクッキーの属性（Secure / SameSite / Domain）の上書き設定を保持する。
type SecureCookie struct {
	Secure   *bool  `env:"SECURE"`
	SameSite string `env:"SAME_SITE"`
	Domain   string `env:"DOMAIN"`
}

// Auth は認証に使う Cookie 名・ヘッダー名・Bearer 許可の設定を保持する。
type Auth struct {
	CookieName          string `env:"COOKIE_NAME,required"`
	HeaderName          string `env:"HEADER_NAME,required"`
	AllowedHeaderBearer bool   `env:"ALLOWED_HEADER_BEARER,required"`
}
