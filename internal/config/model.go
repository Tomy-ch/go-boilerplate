package config

import (
	"net"
	"time"
)

// Config は、アプリケーション全体の設定をサブ設定ごとに束ねたルート設定です。
type Config struct {
	os            OperatingSystemConfig
	app           ApplicationConfig
	server        ServerConfig
	metrics       MetricsConfig
	observability ObservabilityConfig
	database      DatabaseConfig
	dbconnection  DBConnectionConfig
	security      SecurityConfig
	secureCookie  SecureCookieConfig
	worker        WorkerConfig
	consumerQueue ConsumerQueueConfig
	outbox        OutboxConfig
	auth          AuthConfig
	objectStorage ObjectStorageConfig
}

// OperatingSystemConfig は、OS レベルの設定（タイムゾーン）を保持します。
type OperatingSystemConfig struct {
	timezone string
}

// ApplicationConfig は、アプリの識別情報・動作モード・シャットダウン制御の設定を保持します。
type ApplicationConfig struct {
	env             string
	name            string
	mode            string
	logLevel        string
	shutdownTimeout time.Duration
}

// ServerConfig は、HTTP サーバーの待ち受けアドレスと各種タイムアウトを保持します。
type ServerConfig struct {
	host              string
	port              int
	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	bodyLimitMB       int
	requestTimeout    time.Duration
}

// MetricsConfig は、メトリクスエンドポイントの待ち受け情報と認証情報を保持します。
type MetricsConfig struct {
	host     string
	port     int
	userName string
	password string
}

// ObservabilityConfig は、OTLP exporter 設定（trace/metric/log の送出先種別・エンドポイント・プロトコル）と、
// DB クエリ引数マスク・監視対象ステータスコードの設定を保持します。
type ObservabilityConfig struct {
	tracesExporter      string
	metricsExporter     string
	logsExporter        string
	otlpEndpoint        string
	otlpProtocol        string
	maskedDBQueryArgs   bool
	targetStatusCodeSet map[int]bool
}

// DatabaseConfig は、データベースの接続情報とタイムアウト閾値を保持します。
type DatabaseConfig struct {
	driver                 string
	host                   string
	port                   int
	user                   string
	password               string
	name                   string
	sslMode                string
	pingTimeout            time.Duration
	slowQueryWarnThreshold time.Duration
	statementTimeout       time.Duration
	lockTimeout            time.Duration
	txMaxRetries           int
	txRetryBaseBackoff     time.Duration
	txRetryMaxBackoff      time.Duration
}

// DBConnectionConfig は、データベース接続プールのサイズと寿命の設定を保持します。
type DBConnectionConfig struct {
	maxConns    int32
	minConns    int32
	maxLifetime time.Duration
	maxIdleTime time.Duration
}

// SecurityConfig は、CORS・許可 CIDR・セキュリティヘッダー等のセキュリティ設定を保持します。
type SecurityConfig struct {
	allowedOrigins        []string
	cidr                  *net.IPNet
	contentTypeNosniff    string
	xFrameOptions         string
	hstsMaxAge            time.Duration
	hstsExcludeSubdomains bool
	hstsPreloadEnabled    bool
	referrerPolicy        string
}

// SecureCookieConfig は、セキュアクッキーの属性（Secure / SameSite / Domain）の強制設定を保持します。
type SecureCookieConfig struct {
	secure   *bool
	sameSite string
	domain   string
}

// WorkerConfig は、worker engine の engine-core 設定（broker 非依存）を保持します。
type WorkerConfig struct {
	concurrency               int
	maxInFlight               int
	batchSize                 int
	extendInterval            time.Duration
	drainTimeout              time.Duration
	receiveCountWarnThreshold int
	circuitFailureThreshold   int
	circuitOpenBackoffInitial time.Duration
	circuitOpenBackoffMax     time.Duration
	circuitHalfOpenProbe      int
	healthListenAddr          string
	progressStaleAfter        time.Duration
	nackBackoffInitial        time.Duration
	nackBackoffMax            time.Duration
}

// ConsumerQueueConfig は、worker が consume する broker（SQS 互換）の adapter 設定を保持します。
// engine-core の WorkerConfig は broker 非依存のため、broker 語彙を持つ設定は別に保持します。
type ConsumerQueueConfig struct {
	endpoint          string
	region            string
	url               string
	dlqURL            string
	accessKeyID       string
	secretAccessKey   string
	maxMessages       int32
	waitTimeSeconds   int32
	visibilityTimeout int32
}

// OutboxConfig は、transactional outbox relay の設定を保持します。
type OutboxConfig struct {
	publisher            string
	endpoint             string
	pollInterval         time.Duration
	errorBackoff         time.Duration
	batchSize            int
	queueEndpoint        string
	queueRegion          string
	queueURL             string
	queueAccessKeyID     string
	queueSecretAccessKey string
}

// AuthConfig は、access token（JWT）検証の設定を保持します。
// Issuer / Audience / JWKSURL が空の環境では実 JWT authenticator を配線せずスタブが使われます（配線判断は DI）。
type AuthConfig struct {
	issuer             string
	audience           string
	jwksURL            string
	allowedAlgorithms  []string
	clockSkew          time.Duration
	jwksCacheTTL       time.Duration
	discoveryTTL       time.Duration
	unknownKidCooldown time.Duration
}

// ObjectStorageConfig は、S3 互換オブジェクトストレージ（ローカルは Garage）の接続設定と
// アップロード上限を保持します。
type ObjectStorageConfig struct {
	endpoint        string
	region          string
	bucket          string
	accessKeyID     string
	secretAccessKey string
	usePathStyle    bool
	maxUploadBytes  int64
}

// NewOperatingSystemConfig は、OSの設定を返します。
func NewOperatingSystemConfig(cfg *Config) *OperatingSystemConfig { return &cfg.os }

// TimeZone は、OSのタイムゾーンを返します。
func (o *OperatingSystemConfig) TimeZone() string { return o.timezone }

// NewApplicationConfig は、アプリケーションの設定を返します。
func NewApplicationConfig(cfg *Config) *ApplicationConfig { return &cfg.app }

// Env は、デプロイ環境ラベルを返します（自由値）。例: "local" / "staging" / "production"。
func (a *ApplicationConfig) Env() string { return a.env }

// Mode は、アプリの動作モードを返します（"development" / "production" のみ。挙動切替に使用。Env とは別軸）。
func (a *ApplicationConfig) Mode() string { return a.mode }

// LogLevel は、ログ出力レベル（"debug" / "info" / "warn" / "error"）を返します。
func (a *ApplicationConfig) LogLevel() string { return a.logLevel }

// Name は、アプリケーションの名前を返します。
func (a *ApplicationConfig) Name() string { return a.name }

// ShutdownTimeout は、アプリケーションのシャットダウンタイムアウトを返します。
func (a *ApplicationConfig) ShutdownTimeout() time.Duration { return a.shutdownTimeout }

// IsProductionMode は、アプリケーションが本番環境モードかどうかを返します。
func (a *ApplicationConfig) IsProductionMode() bool {
	return a.mode == ProductionMode
}

// IsDevelopmentMode は、アプリケーションが開発環境モードかどうかを返します。
func (a *ApplicationConfig) IsDevelopmentMode() bool {
	return a.mode == DevelopmentMode
}

// NewServerConfig は、サーバーの設定を返します。
func NewServerConfig(cfg *Config) *ServerConfig { return &cfg.server }

// Host は、サーバーがリッスンするホスト名を返します。
func (s *ServerConfig) Host() string { return s.host }

// Port は、サーバーがリッスンするポート番号を返します。
func (s *ServerConfig) Port() int { return s.port }

// ReadHeaderTimeout は、サーバーのヘッダー読み取りタイムアウトを返します。
func (s *ServerConfig) ReadHeaderTimeout() time.Duration { return s.readHeaderTimeout }

// ReadTimeout は、サーバーの読み取りタイムアウトを返します。
func (s *ServerConfig) ReadTimeout() time.Duration { return s.readTimeout }

// WriteTimeout は、サーバーの書き込みタイムアウトを返します。
func (s *ServerConfig) WriteTimeout() time.Duration { return s.writeTimeout }

// IdleTimeout は、サーバーのアイドルタイムアウトを返します。
func (s *ServerConfig) IdleTimeout() time.Duration { return s.idleTimeout }

// BodyLimitMB は、リクエストボディのサイズ上限を MB（10進, 1MB=1,000,000 byte）で返します。
func (s *ServerConfig) BodyLimitMB() int { return s.bodyLimitMB }

// RequestTimeout は、REST リクエスト全体の deadline budget を返します（入口で1点設定し ctx で全層伝播）。
func (s *ServerConfig) RequestTimeout() time.Duration { return s.requestTimeout }

// NewMetricsConfig は、メトリクスの設定を返します。
func NewMetricsConfig(cfg *Config) *MetricsConfig { return &cfg.metrics }

// Host は、メトリクスサーバーがリッスンするホスト名を返します。
func (m *MetricsConfig) Host() string { return m.host }

// Port は、メトリクスサーバーがリッスンするポート番号を返します。
func (m *MetricsConfig) Port() int { return m.port }

// UserName は、メトリクスサーバーの認証に使用するユーザー名を返します。
func (m *MetricsConfig) UserName() string { return m.userName }

// Password は、メトリクスサーバーの認証に使用するパスワードを返します。
func (m *MetricsConfig) Password() string { return m.password }

// NewObservabilityConfig は、可観測の設定を返します。
func NewObservabilityConfig(cfg *Config) *ObservabilityConfig { return &cfg.observability }

// Enabled は、可観測が有効かどうかを返します。
func (o *ObservabilityConfig) Enabled() bool {
	return o.TracesEnabled() || o.MetricsEnabled() || o.LogsEnabled()
}

// TracesEnabled は、trace exporter が有効値かどうかを返します。
func (o *ObservabilityConfig) TracesEnabled() bool { return isActiveExporter(o.tracesExporter) }

// MetricsEnabled は、metric exporter が有効値かどうかを返します。
func (o *ObservabilityConfig) MetricsEnabled() bool { return isActiveExporter(o.metricsExporter) }

// LogsEnabled は、log exporter が有効値かどうかを返します。
func (o *ObservabilityConfig) LogsEnabled() bool { return isActiveExporter(o.logsExporter) }

// OTLPEndpoint は、OTLP exporter の送出先エンドポイントを返します。
func (o *ObservabilityConfig) OTLPEndpoint() string { return o.otlpEndpoint }

// OTLPProtocol は、OTLP exporter のプロトコル（"http/protobuf" / "grpc"）を返します。
func (o *ObservabilityConfig) OTLPProtocol() string { return o.otlpProtocol }

// MaskedDBQueryArgs は、可観測モードでDBクエリの引数をマスクするかどうかを返します。
func (o *ObservabilityConfig) MaskedDBQueryArgs() bool { return o.maskedDBQueryArgs }

// TargetStatusCodeSet は、可観測モードで監視対象となるHTTPステータスコードのセットを返します。
// 返り値はホットパスで参照される共有マップのため変更してはいけません（read-only）。
func (o *ObservabilityConfig) TargetStatusCodeSet() map[int]bool { return o.targetStatusCodeSet }

// NewDatabaseConfig は、データベースの設定を返します。
func NewDatabaseConfig(cfg *Config) *DatabaseConfig { return &cfg.database }

// Driver は、データベースのドライバー名を返します。
func (d *DatabaseConfig) Driver() string { return d.driver }

// Host は、データベースのホスト名を返します。
func (d *DatabaseConfig) Host() string { return d.host }

// Port は、データベースのポート番号を返します。
func (d *DatabaseConfig) Port() int { return d.port }

// User は、データベースのユーザー名を返します。
func (d *DatabaseConfig) User() string { return d.user }

// Password は、データベースのパスワードを返します。
func (d *DatabaseConfig) Password() string { return d.password }

// DBName は、データベースの名前を返します。
func (d *DatabaseConfig) DBName() string { return d.name }

// SSLMode は、データベースのSSLモードを返します。
func (d *DatabaseConfig) SSLMode() string { return d.sslMode }

// PingTimeout は、データベースのpingタイムアウトを返します。
func (d *DatabaseConfig) PingTimeout() time.Duration { return d.pingTimeout }

// SlowQueryWarnThreshold は、スロークエリ警告の閾値を返します。
//
// この値より長く実行されたクエリは警告レベルでログ出力されます。
// 0以下の値の場合、スロークエリ警告は無効になります。
func (d *DatabaseConfig) SlowQueryWarnThreshold() time.Duration { return d.slowQueryWarnThreshold }

// StatementTimeout は、SQL 文の実行時間上限を返します（0 以下で無効）。ctx を無視する runaway query の backstop。
func (d *DatabaseConfig) StatementTimeout() time.Duration { return d.statementTimeout }

// LockTimeout は、ロック獲得待ちの上限を返します（0 以下で無効）。長時間ロック待ちの backstop。
func (d *DatabaseConfig) LockTimeout() time.Duration { return d.lockTimeout }

// TxMaxRetries は、トランザクションのリトライ最大試行回数を返します。
//
// serialization failure / deadlock 検出時の有限リトライ上限です。
// 0 以下の場合は実装側の既定値（3回）にフォールバックします。
func (d *DatabaseConfig) TxMaxRetries() int { return d.txMaxRetries }

// TxRetryBaseBackoff は、トランザクションリトライ backoff の初期値（指数 backoff の基準値）を返します。
// 0 以下の場合は実装側の既定値にフォールバックします。
func (d *DatabaseConfig) TxRetryBaseBackoff() time.Duration { return d.txRetryBaseBackoff }

// TxRetryMaxBackoff は、トランザクションリトライ backoff の上限値（1試行あたりの最大待機時間）を返します。
// 0 以下の場合は実装側の既定値にフォールバックします。
func (d *DatabaseConfig) TxRetryMaxBackoff() time.Duration { return d.txRetryMaxBackoff }

// NewDBConnectionConfig は、データベース接続の設定を返します。
func NewDBConnectionConfig(cfg *Config) *DBConnectionConfig { return &cfg.dbconnection }

// MaxConns は、データベースの最大オープン接続数を返します。
func (c *DBConnectionConfig) MaxConns() int32 { return c.maxConns }

// MinConns は、pgxpool の最小プールサイズ（最小接続数）を返します。
func (c *DBConnectionConfig) MinConns() int32 { return c.minConns }

// MaxLifetime は、データベースの接続の最大寿命を返します。
func (c *DBConnectionConfig) MaxLifetime() time.Duration { return c.maxLifetime }

// MaxIdleTime は、データベースの接続の最大アイドル時間を返します。
func (c *DBConnectionConfig) MaxIdleTime() time.Duration { return c.maxIdleTime }

// NewSecurityConfig は、セキュリティの設定を返します。
func NewSecurityConfig(cfg *Config) *SecurityConfig { return &cfg.security }

// AllowedOrigins は、CORSを許可するオリジンのリストを返します。
func (s *SecurityConfig) AllowedOrigins() []string {
	return append([]string(nil), s.allowedOrigins...)
}

// CIDR は、セキュリティ設定で使用されるCIDRを返します。
func (s *SecurityConfig) CIDR() *net.IPNet {
	if s.cidr == nil {
		return nil
	}
	return &net.IPNet{
		IP:   append(net.IP(nil), s.cidr.IP...),
		Mask: append(net.IPMask(nil), s.cidr.Mask...),
	}
}

// ContentTypeNosniff は、X-Content-Type-Optionsヘッダーの値を返します。
func (s *SecurityConfig) ContentTypeNosniff() string { return s.contentTypeNosniff }

// XFrameOptions は、X-Frame-Optionsヘッダーの値を返します。
func (s *SecurityConfig) XFrameOptions() string { return s.xFrameOptions }

// HSTSMaxAge は、HSTSの最大年齢を返します。
func (s *SecurityConfig) HSTSMaxAge() time.Duration { return s.hstsMaxAge }

// HSTSExcludeSubdomains は、HSTSでサブドメインを除外するかどうかを返します。
func (s *SecurityConfig) HSTSExcludeSubdomains() bool { return s.hstsExcludeSubdomains }

// HSTSPreloadEnabled は、HSTSのプリロードが有効かどうかを返します。
func (s *SecurityConfig) HSTSPreloadEnabled() bool { return s.hstsPreloadEnabled }

// ReferrerPolicy は、Referrer-Policyヘッダーの値を返します。
func (s *SecurityConfig) ReferrerPolicy() string { return s.referrerPolicy }

// NewSecureCookieConfig は、セキュアクッキーの設定を返します。
func NewSecureCookieConfig(cfg *Config) *SecureCookieConfig { return &cfg.secureCookie }

// Secure は、Secure 属性の強制設定を返します。nil は「上書きしない」ことを意味します。
func (s *SecureCookieConfig) Secure() *bool {
	if s.secure == nil {
		return nil
	}
	v := *s.secure
	return &v
}

// SameSite は、SameSite 属性の強制設定を返します。空文字列は「上書きしない」ことを意味します。
func (s *SecureCookieConfig) SameSite() string { return s.sameSite }

// Domain は、Domain 属性の強制設定を返します。空文字列は「上書きしない」ことを意味します。
func (s *SecureCookieConfig) Domain() string { return s.domain }

// NewWorkerConfig は、worker engine の設定を返します。
func NewWorkerConfig(cfg *Config) *WorkerConfig { return &cfg.worker }

// Concurrency は、同時に Handle を実行する最大数を返します。
func (w *WorkerConfig) Concurrency() int { return w.concurrency }

// MaxInFlight は、受信済み・未確定の最大メッセージ数を返します。
func (w *WorkerConfig) MaxInFlight() int { return w.maxInFlight }

// BatchSize は、1 回の Receive で取得する最大件数を返します。
func (w *WorkerConfig) BatchSize() int { return w.batchSize }

// ExtendInterval は、Extend を呼ぶ周期を返します（0 以下で無効）。
func (w *WorkerConfig) ExtendInterval() time.Duration { return w.extendInterval }

// DrainTimeout は、停止時に in-flight の完了を待つ上限を返します。
func (w *WorkerConfig) DrainTimeout() time.Duration { return w.drainTimeout }

// ReceiveCountWarnThreshold は、再配送回数の警告閾値を返します（0 以下で無効）。
func (w *WorkerConfig) ReceiveCountWarnThreshold() int { return w.receiveCountWarnThreshold }

// CircuitFailureThreshold は、サーキットを Open にする連続失敗数を返します（0 以下で無効）。
func (w *WorkerConfig) CircuitFailureThreshold() int { return w.circuitFailureThreshold }

// CircuitOpenBackoffInitial は、Open の初回 cooldown を返します。
func (w *WorkerConfig) CircuitOpenBackoffInitial() time.Duration { return w.circuitOpenBackoffInitial }

// CircuitOpenBackoffMax は、Open の cooldown 上限を返します。
func (w *WorkerConfig) CircuitOpenBackoffMax() time.Duration { return w.circuitOpenBackoffMax }

// CircuitHalfOpenProbe は、Half-open 時に試行する最大件数を返します。
func (w *WorkerConfig) CircuitHalfOpenProbe() int { return w.circuitHalfOpenProbe }

// HealthListenAddr は、liveness/readiness を公開する health listener の待ち受けアドレスを返します。
func (w *WorkerConfig) HealthListenAddr() string { return w.healthListenAddr }

// ProgressStaleAfter は、readiness 判定で「進捗なし」とみなすまでの時間を返します。
func (w *WorkerConfig) ProgressStaleAfter() time.Duration { return w.progressStaleAfter }

// NackBackoffInitial は、retryable 失敗時の per-message 再配送 backoff の初回待機を返します。
func (w *WorkerConfig) NackBackoffInitial() time.Duration { return w.nackBackoffInitial }

// NackBackoffMax は、per-message 再配送 backoff の上限を返します。
func (w *WorkerConfig) NackBackoffMax() time.Duration { return w.nackBackoffMax }

// NewConsumerQueueConfig は、worker が consume する broker の adapter 設定を返します。
func NewConsumerQueueConfig(cfg *Config) *ConsumerQueueConfig { return &cfg.consumerQueue }

// Endpoint は、SQS 互換エンドポイントを返します（空なら SDK 既定の解決に委ねます）。
func (c *ConsumerQueueConfig) Endpoint() string { return c.endpoint }

// Region は、SQS の署名に用いるリージョンを返します。
func (c *ConsumerQueueConfig) Region() string { return c.region }

// URL は、consume 対象キューの URL を返します。
func (c *ConsumerQueueConfig) URL() string { return c.url }

// DLQURL は、滞留量の収集対象とする DLQ の URL を返します（空なら収集しません）。
func (c *ConsumerQueueConfig) DLQURL() string { return c.dlqURL }

// AccessKeyID は、明示注入する静的資格情報のアクセスキー ID を返します（空なら chain へ委ねます）。
func (c *ConsumerQueueConfig) AccessKeyID() string { return c.accessKeyID }

// SecretAccessKey は、明示注入する静的資格情報のシークレットアクセスキーを返します（空なら chain へ委ねます）。
func (c *ConsumerQueueConfig) SecretAccessKey() string { return c.secretAccessKey }

// MaxMessages は、1 回の受信で取得する最大件数を返します。
func (c *ConsumerQueueConfig) MaxMessages() int32 { return c.maxMessages }

// WaitTimeSeconds は、long-poll の待機秒数を返します。
func (c *ConsumerQueueConfig) WaitTimeSeconds() int32 { return c.waitTimeSeconds }

// VisibilityTimeout は、受信メッセージの可視性タイムアウト秒数を返します。
func (c *ConsumerQueueConfig) VisibilityTimeout() int32 { return c.visibilityTimeout }

// NewOutboxConfig は、outbox relay の設定を返します。
func NewOutboxConfig(cfg *Config) *OutboxConfig { return &cfg.outbox }

// Publisher は、publish 先の種別（"http" / "sqs"）を返します。
func (o *OutboxConfig) Publisher() string { return o.publisher }

// Endpoint は、メッセージの送信先エンドポイント URL を返します。
func (o *OutboxConfig) Endpoint() string { return o.endpoint }

// QueueEndpoint は、SQS 互換エンドポイントを返します（空なら SDK 既定の解決に委ねます）。
func (o *OutboxConfig) QueueEndpoint() string { return o.queueEndpoint }

// QueueRegion は、SQS の署名に用いるリージョンを返します。
func (o *OutboxConfig) QueueRegion() string { return o.queueRegion }

// QueueURL は、publish 先キューの URL を返します。
func (o *OutboxConfig) QueueURL() string { return o.queueURL }

// QueueAccessKeyID は、SQS の静的資格情報のアクセスキー ID を返します。
func (o *OutboxConfig) QueueAccessKeyID() string { return o.queueAccessKeyID }

// QueueSecretAccessKey は、SQS の静的資格情報のシークレットアクセスキーを返します。
func (o *OutboxConfig) QueueSecretAccessKey() string { return o.queueSecretAccessKey }

// PollInterval は、pending を捌き切った後に次 poll まで待機する時間を返します。
func (o *OutboxConfig) PollInterval() time.Duration { return o.pollInterval }

// ErrorBackoff は、relay バッチがエラーを返した後に待機する時間を返します。
func (o *OutboxConfig) ErrorBackoff() time.Duration { return o.errorBackoff }

// BatchSize は、1 回の poll で claim する pending 行数を返します。
func (o *OutboxConfig) BatchSize() int { return o.batchSize }

// NewAuthConfig は、認証（JWT 検証）の設定を返します。
func NewAuthConfig(cfg *Config) *AuthConfig { return &cfg.auth }

// Issuer は、検証する iss クレームの期待値を返します（空なら実 JWT authenticator を配線しない）。
func (a *AuthConfig) Issuer() string { return a.issuer }

// Audience は、検証する aud クレームの期待値を返します。
func (a *AuthConfig) Audience() string { return a.audience }

// JWKSURL は、公開鍵を取得する JWKS エンドポイント URL を返します。
func (a *AuthConfig) JWKSURL() string { return a.jwksURL }

// AllowedAlgorithms は、許可する署名アルゴリズムの allowlist を返します。
func (a *AuthConfig) AllowedAlgorithms() []string {
	return append([]string(nil), a.allowedAlgorithms...)
}

// ClockSkew は、exp / nbf 検証時のクロックずれ許容幅を返します。
func (a *AuthConfig) ClockSkew() time.Duration { return a.clockSkew }

// JWKSCacheTTL は、取得した JWKS をキャッシュする期間を返します。
func (a *AuthConfig) JWKSCacheTTL() time.Duration { return a.jwksCacheTTL }

// DiscoveryTTL は、OIDC discovery 文書の再取得間隔を返します。
func (a *AuthConfig) DiscoveryTTL() time.Duration { return a.discoveryTTL }

// UnknownKidCooldown は、未知 kid での JWKS 再取得の最小間隔を返します。
func (a *AuthConfig) UnknownKidCooldown() time.Duration { return a.unknownKidCooldown }

// NewObjectStorageConfig は、オブジェクトストレージの設定を返します。
func NewObjectStorageConfig(cfg *Config) *ObjectStorageConfig { return &cfg.objectStorage }

// Endpoint は、S3 互換エンドポイント URL を返します（空なら SDK 既定のエンドポイント解決に委ねます）。
func (o *ObjectStorageConfig) Endpoint() string { return o.endpoint }

// Region は、署名に用いるリージョンを返します。
func (o *ObjectStorageConfig) Region() string { return o.region }

// Bucket は、オブジェクトを格納するバケット名を返します。
func (o *ObjectStorageConfig) Bucket() string { return o.bucket }

// AccessKeyID は、静的資格情報のアクセスキー ID を返します。
func (o *ObjectStorageConfig) AccessKeyID() string { return o.accessKeyID }

// SecretAccessKey は、静的資格情報のシークレットアクセスキーを返します。
func (o *ObjectStorageConfig) SecretAccessKey() string { return o.secretAccessKey }

// UsePathStyle は、path-style アクセスを使うかどうかを返します（Garage / MinIO は true）。
func (o *ObjectStorageConfig) UsePathStyle() bool { return o.usePathStyle }

// MaxUploadBytes は、アップロード可能な最大バイト数を返します。
// ServerConfig.BodyLimitMB（マルチパートのオーバーヘッドを含む）を上回る値を設定すると、
// グローバルな body limit が先に 413 を返すためこの上限は到達不能になります。
// この関係は server グラフの起動時に ValidateUploadBodyLimit が検証します。
func (o *ObjectStorageConfig) MaxUploadBytes() int64 { return o.maxUploadBytes }
