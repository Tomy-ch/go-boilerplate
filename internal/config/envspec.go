package config

import "time"

// Loader はアプリケーション全体の設定を環境変数から読み込むルートコンテナ。
type Loader struct {
	OS            OperatingSystem `envPrefix:"OS_"`
	App           Application     `envPrefix:"APP_"`
	Server        Server          `envPrefix:"SERVER_"`
	Observability Observability   `envPrefix:"OBS_"`
	Metrics       Metrics         `envPrefix:"METRICS_"`
	Database      Database        `envPrefix:"DB_"`
	DBConnection  DBConnection    `envPrefix:"DBCONN_"`
	Security      Security        `envPrefix:"SECURITY_"`
	SecureCookie  SecureCookie    `envPrefix:"SECURE_COOKIE_"`
	Worker        Worker          `envPrefix:"WORKER_"`
	Outbox        Outbox          `envPrefix:"OUTBOX_"`
	Auth          Auth            `envPrefix:"AUTH_"`
}

// Auth は access token（JWT）検証の設定を保持する。
type Auth struct {
	Issuer            string        `env:"ISSUER"             envDefault:""`
	Audience          string        `env:"AUDIENCE"           envDefault:""`
	JWKSURL           string        `env:"JWKS_URL"           envDefault:""`
	AllowedAlgorithms []string      `env:"ALLOWED_ALGORITHMS" envDefault:"RS256" envSeparator:","`
	ClockSkew         time.Duration `env:"CLOCK_SKEW"         envDefault:"60s"`
	JWKSCacheTTL      time.Duration `env:"JWKS_CACHE_TTL"     envDefault:"1h"`
	// JWKSDiscoveryTTL は OIDC discovery 文書の再取得間隔です（鍵キャッシュとは別軸）。
	JWKSDiscoveryTTL time.Duration `env:"JWKS_DISCOVERY_TTL" envDefault:"24h"`
	// JWKSUnknownKIDCooldown は未知 kid での JWKS 再取得の最小間隔です（DoS 抑止）。
	JWKSUnknownKIDCooldown time.Duration `env:"JWKS_UNKNOWN_KID_COOLDOWN" envDefault:"60s"`
}

// Outbox は transactional outbox relay の設定を保持する。
type Outbox struct {
	Endpoint     string        `env:"ENDPOINT"      envDefault:""`
	PollInterval time.Duration `env:"POLL_INTERVAL" envDefault:"1s"`
	ErrorBackoff time.Duration `env:"ERROR_BACKOFF" envDefault:"5s"`
	BatchSize    int           `env:"BATCH_SIZE"    envDefault:"100"`
}

// Worker は worker engine の engine-core 設定（broker 非依存）を保持する。
type Worker struct {
	Concurrency               int           `env:"CONCURRENCY"                  envDefault:"4"`
	MaxInFlight               int           `env:"MAX_IN_FLIGHT"                envDefault:"8"`
	BatchSize                 int           `env:"BATCH_SIZE"                   envDefault:"4"`
	ExtendInterval            time.Duration `env:"EXTEND_INTERVAL"              envDefault:"0s"`
	DrainTimeout              time.Duration `env:"DRAIN_TIMEOUT"                envDefault:"30s"`
	ReceiveCountWarnThreshold int           `env:"RECEIVE_COUNT_WARN_THRESHOLD" envDefault:"5"`
	CircuitFailureThreshold   int           `env:"CIRCUIT_FAILURE_THRESHOLD"    envDefault:"10"`
	CircuitOpenBackoffInitial time.Duration `env:"CIRCUIT_OPEN_BACKOFF_INITIAL" envDefault:"1s"`
	CircuitOpenBackoffMax     time.Duration `env:"CIRCUIT_OPEN_BACKOFF_MAX"     envDefault:"30s"`
	CircuitHalfOpenProbe      int           `env:"CIRCUIT_HALF_OPEN_PROBE"      envDefault:"1"`
	HealthListenAddr          string        `env:"HEALTH_LISTEN_ADDR"           envDefault:":8081"`
	ProgressStaleAfter        time.Duration `env:"PROGRESS_STALE_AFTER"         envDefault:"60s"`
	NackBackoffInitial        time.Duration `env:"NACK_BACKOFF_INITIAL"         envDefault:"1s"`
	NackBackoffMax            time.Duration `env:"NACK_BACKOFF_MAX"             envDefault:"30s"`
}

// OperatingSystem は OS レベルの設定を保持する。
type OperatingSystem struct {
	Timezone string `env:"TZ" envDefault:"Asia/Tokyo"`
}

// Application はアプリケーション識別・動作モードおよびシャットダウン制御に関する設定を保持する。
type Application struct {
	Env             string        `env:"ENV,required"`
	Name            string        `env:"NAME,required"`
	Mode            string        `env:"MODE,required"`
	LogLevel        string        `env:"LOG_LEVEL,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT"   envDefault:"65s"`
}

// Server は HTTP サーバーのバインドアドレスおよび各種タイムアウトに関する設定を保持する。
type Server struct {
	Host              string        `env:"HOST,required"`
	Port              int           `env:"PORT,required"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT"        envDefault:"10s"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT"       envDefault:"65s"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT"        envDefault:"60s"`
	BodyLimitMB       int           `env:"BODY_LIMIT_MB"       envDefault:"5"`
	RequestTimeout    time.Duration `env:"REQUEST_TIMEOUT"     envDefault:"60s"`
}

// Metrics はメトリクスエンドポイント（Prometheus 等）への接続情報と認証情報を保持する。
type Metrics struct {
	Host     string `env:"HOST,required"`
	Port     int    `env:"PORT,required"`
	UserName string `env:"USERNAME,required,notEmpty"`
	Password string `env:"PASSWORD,required,notEmpty"`
}

// Observability は OTLP exporter 設定（trace/metric/log の送出先種別・エンドポイント・プロトコル）、
// DBクエリ引数マスク設定、およびトレース対象とする HTTP ステータスコード一覧を保持する。
type Observability struct {
	TracesExporter    string `env:"TRACES_EXPORTER"`
	MetricsExporter   string `env:"METRICS_EXPORTER"`
	LogsExporter      string `env:"LOGS_EXPORTER"`
	OTLPEndpoint      string `env:"OTLP_ENDPOINT"`
	OTLPProtocol      string `env:"OTLP_PROTOCOL"                 envDefault:"http/protobuf"`
	MaskedDBQueryArgs bool   `env:"MASKED_DB_QUERY_ARGS,required"`
	TargetStatusCodes []int  `env:"TARGET_STATUS_CODES,required"                             envSeparator:","`
}

// Database はデータベースへの接続先情報（ドライバ・ホスト・認証情報・SSL）および
// 接続確認タイムアウトやスロークエリ警告閾値を保持する。
type Database struct {
	Driver                 string        `env:"DRIVER"                    envDefault:"pgx"`
	Host                   string        `env:"HOST,required"`
	Port                   int           `env:"PORT,required"`
	User                   string        `env:"USER,required"`
	Password               string        `env:"PASSWORD,required"`
	Name                   string        `env:"NAME,required"`
	SSLMode                string        `env:"SSL_MODE,required"`
	PingTimeout            time.Duration `env:"PING_TIMEOUT,required"`
	SlowQueryWarnThreshold time.Duration `env:"SLOW_QUERY_WARN_THRESHOLD" envDefault:"500ms"`
	StatementTimeout       time.Duration `env:"STATEMENT_TIMEOUT"         envDefault:"30s"`
	LockTimeout            time.Duration `env:"LOCK_TIMEOUT"              envDefault:"10s"`
	TxMaxRetries           int           `env:"TX_MAX_RETRIES"            envDefault:"3"`
	TxRetryBaseBackoff     time.Duration `env:"TX_RETRY_BASE_BACKOFF"     envDefault:"5ms"`
	TxRetryMaxBackoff      time.Duration `env:"TX_RETRY_MAX_BACKOFF"      envDefault:"100ms"`
}

// DBConnection はコネクションプールの上限・下限数とコネクションの最大生存時間・最大アイドル時間を保持する。
type DBConnection struct {
	MaxConns    int32         `env:"MAX_CONNS"     envDefault:"10"`
	MinConns    int32         `env:"MIN_CONNS"     envDefault:"5"`
	MaxLifetime time.Duration `env:"MAX_LIFETIME"  envDefault:"30m"`
	MaxIdleTime time.Duration `env:"MAX_IDLE_TIME" envDefault:"10m"`
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
