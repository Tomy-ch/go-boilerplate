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
	ConsumerQueue ConsumerQueue   `envPrefix:"CONSUMER_QUEUE_"`
	Outbox        Outbox          `envPrefix:"OUTBOX_"`
	Auth          Auth            `envPrefix:"AUTH_"`
	ObjectStorage ObjectStorage   `envPrefix:"OBJECT_STORAGE_"`
	Realtime      Realtime        `envPrefix:"REALTIME_"`
	Endpoint      Endpoint        `envPrefix:"ENDPOINT_"`
}

// Endpoint は、このアプリが接続する外部サービスの所在をまとめて保持する。
// 「どこへ繋ぐか」はデプロイごとに変わる軸であり、各サブシステムの「どう振る舞うか」とは
// 直交するため、サブシステム設定から切り離してここへ集める。
// 全項目 required だが空文字を許容する。空の意味は接続先ごとに異なるため各フィールドに記す。
type Endpoint struct {
	// OTLP は OpenTelemetry Collector の送出先です。空なら送出しません。
	OTLP string `env:"OTLP,required"`
	// JWKS は公開鍵の取得先です。空なら OIDC discovery で issuer から解決します。
	JWKS string `env:"JWKS,required"`
	// ObjectStorage は S3 互換エンドポイントです。空なら SDK 既定の解決に委ねます（本番 AWS S3 等）。
	ObjectStorage string `env:"OBJECT_STORAGE,required"`
	// Realtime は Realtime Delivery の store（DynamoDB 互換）のエンドポイントです。空なら SDK 既定の解決に委ねます（本番 DynamoDB）。
	Realtime string `env:"REALTIME,required"`
	// RealtimePubSub は Realtime Delivery の fan-out（SNS / SQS 互換）のエンドポイントです。空なら SDK 既定の解決に委ねます（本番 SNS / SQS）。
	RealtimePubSub string `env:"REALTIME_PUBSUB,required"`
	// Outbox は PUBLISHER=http のときの送出先です。空のまま http を選ぶと DI で起動エラーにします。
	Outbox string `env:"OUTBOX,required"`
	// OutboxQueue は publish 端の SQS 互換エンドポイントです。空なら SDK 既定の解決に委ねます。
	OutboxQueue string `env:"OUTBOX_QUEUE,required"`
	// ConsumerQueue は consume 端の SQS 互換エンドポイントです。空なら SDK 既定の解決に委ねます。
	ConsumerQueue string `env:"CONSUMER_QUEUE,required"`
	// sample-api:begin
	// ExchangeRate は外部為替レートサービスのベース URL です。空ならこの機能を使いません。
	ExchangeRate string `env:"EXCHANGE_RATE,required"`
	// sample-api:end
}

// ObjectStorage は、画像等を格納する S3 互換オブジェクトストレージ（ローカルは Garage）の接続設定と
// アップロード上限を保持する。中立境界の実装は S3 アダプタだが、env 名は vendor 非依存にする。
type ObjectStorage struct {
	Region string `env:"REGION,required,notEmpty"`
	Bucket string `env:"BUCKET,required,notEmpty"`
	// AccessKeyID / SecretAccessKey は両方空なら SDK 既定の credential chain（IAM ロール等）へ委ねるため、
	// 未設定を許す。ロール運用のデプロイにダミー値の注入を強いないための既定。
	AccessKeyID     string `env:"ACCESS_KEY_ID"                      envDefault:""`
	SecretAccessKey string `env:"SECRET_ACCESS_KEY"                  envDefault:""`
	UsePathStyle    bool   `env:"USE_PATH_STYLE,required"`
	MaxUploadBytes  int64  `env:"MAX_UPLOAD_BYTES,required,notEmpty"`
}

// Realtime は、Realtime Delivery の EventLog / StreamTicket / InstanceLease を置く DynamoDB 互換 store と、
// serve instance への fan-out（SNS / SQS 互換）の接続設定を保持する。中立境界の実装は DynamoDB / SNS / SQS の
// adapter だが、env 名は vendor 非依存にする。
// table 名は固定名（realtime_event_log 等）に TableSuffix を付けた形で、環境（と Phase 11 では worktree の
// slot）ごとに分かれる。instance queue の名前は QueuePrefix に instance の識別子を付けた形で、環境ごとに分かれる。
type Realtime struct {
	Region string `env:"REGION,required,notEmpty"`
	// TableSuffix は table 名の末尾に付く環境識別子（例 local / ci / prd）。小文字・数字・アンダースコアで書く。
	TableSuffix string `env:"TABLE_SUFFIX,required,notEmpty"`
	// Topic は wakeup と失効通知を載せる topic の識別子（AWS では ARN）。deployment が用意する resource で、
	// 空のまま fan-out を配線すると DI で起動エラーにする（ENDPOINT_OUTBOX と同じ扱い）。
	Topic string `env:"TOPIC" envDefault:""`
	// QueuePrefix は instance queue 名の先頭（例 realtime-local）。英数字・`-`・`_` で書く。
	QueuePrefix string `env:"QUEUE_PREFIX,required,notEmpty"`
	// DLQ は instance queue の redrive 先の識別子（AWS では ARN）。空なら RedrivePolicy を付けない。
	DLQ string `env:"DLQ" envDefault:""`
	// AccessKeyID / SecretAccessKey は両方空なら SDK 既定の credential chain（IAM ロール等）へ委ねるため、
	// 未設定を許す（ObjectStorage と同じ既定）。
	AccessKeyID     string `env:"ACCESS_KEY_ID"     envDefault:""`
	SecretAccessKey string `env:"SECRET_ACCESS_KEY" envDefault:""`
}

// Auth は access token（JWT）検証の設定を保持する。
type Auth struct {
	Issuer            string        `env:"ISSUER"             envDefault:""`
	Audience          string        `env:"AUDIENCE"           envDefault:""`
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
	// Publisher は publish 先の種別（"http" / "sqs"）です。ENV ではなくこの判別子で切り替え、
	// 未知の値は DI で起動エラーにする（fail-closed）。
	Publisher    string        `env:"PUBLISHER"     envDefault:"http"`
	PollInterval time.Duration `env:"POLL_INTERVAL" envDefault:"1s"`
	ErrorBackoff time.Duration `env:"ERROR_BACKOFF" envDefault:"5s"`
	BatchSize    int           `env:"BATCH_SIZE"    envDefault:"100"`
	// Queue* は PUBLISHER=sqs のときだけ使う。未設定のまま sqs を選ぶと adapter 構築時に落とす。
	QueueRegion          string `env:"QUEUE_REGION"            envDefault:""`
	QueueURL             string `env:"QUEUE_URL"               envDefault:""`
	QueueAccessKeyID     string `env:"QUEUE_ACCESS_KEY_ID"     envDefault:""`
	QueueSecretAccessKey string `env:"QUEUE_SECRET_ACCESS_KEY" envDefault:""`
}

// ConsumerQueue は worker が consume する broker（SQS 互換）の adapter 設定を保持する。
// engine-core の Worker は broker 非依存と定めているため、broker 語彙を持つ設定は別軸に置く。
// publish 端の Outbox.Queue* とは対になる（consume 端がこちら）。
// 資格情報が両方空なら SDK 既定の credential chain（IAM ロール等）へ委ねる。
type ConsumerQueue struct {
	Region          string `env:"REGION"            envDefault:""`
	URL             string `env:"URL"               envDefault:""`
	AccessKeyID     string `env:"ACCESS_KEY_ID"     envDefault:""`
	SecretAccessKey string `env:"SECRET_ACCESS_KEY" envDefault:""`
	// DLQURL は Permanent メッセージの退避先であり、滞留量の収集対象でもある。
	// 空なら app 側の退避経路を持たず、broker の redrive policy に委ねる。
	DLQURL string `env:"DLQ_URL" envDefault:""`
	// MaxMessages は ReceiveMessage の最大取得件数（SQS の上限は 10）。
	MaxMessages int32 `env:"MAX_MESSAGES" envDefault:"10"`
	// WaitTimeSeconds は long-poll の待機秒数（0〜20）。上限にすると空ポーリングの往復が最も減る。
	WaitTimeSeconds int32 `env:"WAIT_TIME_SECONDS" envDefault:"20"`
	// VisibilityTimeout は受信メッセージの可視性タイムアウト秒数。
	VisibilityTimeout int32 `env:"VISIBILITY_TIMEOUT" envDefault:"30"`
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
	Timezone string `env:"TZ,required,notEmpty"`
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
	BodyLimitMB       int           `env:"BODY_LIMIT_MB"       envDefault:"6"`
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
// Referrer Policy を保持する。
type Security struct {
	AllowedOrigins        []string      `env:"ALLOWED_ORIGINS,required"         envSeparator:","`
	CIDR                  string        `env:"CIDR,required"`
	ContentTypeNosniff    string        `env:"CONTENT_TYPE_NOSNIFF,required"`
	XFrameOptions         string        `env:"X_FRAME_OPTIONS,required"`
	HSTSMaxAge            time.Duration `env:"HSTS_MAX_AGE,required"`
	HSTSExcludeSubdomains bool          `env:"HSTS_EXCLUDE_SUBDOMAINS,required"`
	HSTSPreloadEnabled    bool          `env:"HSTS_PRELOAD_ENABLED,required"`
	ReferrerPolicy        string        `env:"REFERRER_POLICY,required"`
	// CrossOriginResourcePolicy は Cross-Origin-Resource-Policy ヘッダーの値です。空の場合は
	// ヘッダーを出さない（ブラウザ既定の挙動に委ねる）という意味を持つため、空文字を許容する。
	CrossOriginResourcePolicy string `env:"CROSS_ORIGIN_RESOURCE_POLICY,required"`
}

// SecureCookie はセキュアクッキーの属性（Secure / SameSite / Domain）の上書き設定を保持する。
type SecureCookie struct {
	Secure   *bool  `env:"SECURE"`
	SameSite string `env:"SAME_SITE"`
	Domain   string `env:"DOMAIN"`
}
