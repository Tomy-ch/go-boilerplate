package config

import (
	"net"
	"testing"
	"time"
)

// WARN: 本番コードでは使用しないでください。テスト用の設定を行うためのメソッドです。

// SetApplicationMode は、テスト用にサーバーのAppModeを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *ApplicationConfig) SetApplicationMode(tb testing.TB, mode string) {
	tb.Helper()
	prev := a.Mode()
	a.mode = mode
	tb.Cleanup(func() { a.mode = prev })
}

// SetApplicationEnv は、テスト用にアプリケーションの環境を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *ApplicationConfig) SetApplicationEnv(tb testing.TB, env string) {
	tb.Helper()
	prev := a.Env()
	a.env = env
	tb.Cleanup(func() { a.env = prev })
}

// SetApplicationLogLevel は、テスト用にアプリケーションのログレベルを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *ApplicationConfig) SetApplicationLogLevel(tb testing.TB, level string) {
	tb.Helper()
	prev := a.LogLevel()
	a.logLevel = level
	tb.Cleanup(func() { a.logLevel = prev })
}

// SetServerPort は、テスト用にサーバーのポートを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (s *ServerConfig) SetServerPort(tb testing.TB, port int) {
	tb.Helper()
	prev := s.Port()
	s.port = port
	tb.Cleanup(func() { s.port = prev })
}

// SetObservabilityMaskedDBQueryArgs は、テスト用にオブザーバビリティのDBクエリ引数の設定を行います。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *ObservabilityConfig) SetObservabilityMaskedDBQueryArgs(tb testing.TB, val bool) {
	tb.Helper()
	prev := o.MaskedDBQueryArgs()
	o.maskedDBQueryArgs = val
	tb.Cleanup(func() { o.maskedDBQueryArgs = prev })
}

// SetObservabilityTracesExporter は、テスト用に trace exporter 指定を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *ObservabilityConfig) SetObservabilityTracesExporter(tb testing.TB, val string) {
	tb.Helper()
	prev := o.tracesExporter
	o.tracesExporter = val
	tb.Cleanup(func() { o.tracesExporter = prev })
}

// SetObservabilityMetricsExporter は、テスト用に metric exporter 指定を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *ObservabilityConfig) SetObservabilityMetricsExporter(tb testing.TB, val string) {
	tb.Helper()
	prev := o.metricsExporter
	o.metricsExporter = val
	tb.Cleanup(func() { o.metricsExporter = prev })
}

// SetObservabilityLogsExporter は、テスト用に log exporter 指定を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *ObservabilityConfig) SetObservabilityLogsExporter(tb testing.TB, val string) {
	tb.Helper()
	prev := o.logsExporter
	o.logsExporter = val
	tb.Cleanup(func() { o.logsExporter = prev })
}

// SetObservabilityOTLPProtocol は、テスト用に OTLP プロトコル指定を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *ObservabilityConfig) SetObservabilityOTLPProtocol(tb testing.TB, val string) {
	tb.Helper()
	prev := o.otlpProtocol
	o.otlpProtocol = val
	tb.Cleanup(func() { o.otlpProtocol = prev })
}

// SetObservabilityOTLPEndpoint は、テスト用に OTLP エンドポイント指定を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *ObservabilityConfig) SetObservabilityOTLPEndpoint(tb testing.TB, val string) {
	tb.Helper()
	prev := o.otlpEndpoint
	o.otlpEndpoint = val
	tb.Cleanup(func() { o.otlpEndpoint = prev })
}

// SetDatabaseHost は、テスト用にデータベースのホスト名を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (d *DatabaseConfig) SetDatabaseHost(tb testing.TB, host string) {
	tb.Helper()
	prev := d.Host()
	d.host = host
	tb.Cleanup(func() { d.host = prev })
}

// SetDatabaseName は、テスト用にデータベースの名前を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (d *DatabaseConfig) SetDatabaseName(tb testing.TB, name string) {
	tb.Helper()
	prev := d.DBName()
	d.name = name
	tb.Cleanup(func() { d.name = prev })
}

// SetMetricsPort は、テスト用にメトリクスサーバーのポートを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (m *MetricsConfig) SetMetricsPort(tb testing.TB, port int) {
	tb.Helper()
	prev := m.Port()
	m.port = port
	tb.Cleanup(func() { m.port = prev })
}

// SetHealthListenAddr は、テスト用に worker health listener の待ち受けアドレスを設定します。
//
// 並列サブテストでは固定ポート（既定 :8081）が衝突するため "127.0.0.1:0" を渡して
// OS 割り当ての空きポートを使わせる用途を想定します。t.Setenv と異なり t.Parallel と併用可能です。
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (w *WorkerConfig) SetHealthListenAddr(tb testing.TB, addr string) {
	tb.Helper()
	prev := w.HealthListenAddr()
	w.healthListenAddr = addr
	tb.Cleanup(func() { w.healthListenAddr = prev })
}

// SetMaxConns は、テスト用にDB接続の最大オープン数を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *DBConnectionConfig) SetMaxConns(tb testing.TB, maxConns int32) {
	tb.Helper()
	prev := c.MaxConns()
	c.maxConns = maxConns
	tb.Cleanup(func() { c.maxConns = prev })
}

// SetCIDR は、テスト用にセキュリティのCIDRを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (s *SecurityConfig) SetCIDR(tb testing.TB, cidr *net.IPNet) {
	tb.Helper()
	prev := s.CIDR()
	s.cidr = cidr
	tb.Cleanup(func() { s.cidr = prev })
}

// SetOutboxBatchSize は、テスト用に outbox relay の batch size を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *OutboxConfig) SetOutboxBatchSize(tb testing.TB, batchSize int) {
	tb.Helper()
	prev := o.batchSize
	o.batchSize = batchSize
	tb.Cleanup(func() { o.batchSize = prev })
}

// SetOutboxPollInterval は、テスト用に outbox relay の poll 間隔を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *OutboxConfig) SetOutboxPollInterval(tb testing.TB, pollInterval time.Duration) {
	tb.Helper()
	prev := o.pollInterval
	o.pollInterval = pollInterval
	tb.Cleanup(func() { o.pollInterval = prev })
}

// SetOutboxErrorBackoff は、テスト用に outbox relay のエラー時待機時間を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *OutboxConfig) SetOutboxErrorBackoff(tb testing.TB, errorBackoff time.Duration) {
	tb.Helper()
	prev := o.errorBackoff
	o.errorBackoff = errorBackoff
	tb.Cleanup(func() { o.errorBackoff = prev })
}

// SetOutboxEndpoint は、テスト用に outbox relay の送信先エンドポイントを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *OutboxConfig) SetOutboxEndpoint(tb testing.TB, endpoint string) {
	tb.Helper()
	prev := o.endpoint
	o.endpoint = endpoint
	tb.Cleanup(func() { o.endpoint = prev })
}

// SetOutboxPublisher は、テスト用に outbox の publish 先種別を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *OutboxConfig) SetOutboxPublisher(tb testing.TB, publisher string) {
	tb.Helper()
	prev := o.publisher
	o.publisher = publisher
	tb.Cleanup(func() { o.publisher = prev })
}

// SetOutboxQueue は、テスト用に SQS publish 先の設定をまとめて行います。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *OutboxConfig) SetOutboxQueue(tb testing.TB, queueURL, region, accessKeyID, secretAccessKey string) {
	tb.Helper()
	prev := *o
	o.queueURL, o.queueRegion = queueURL, region
	o.queueAccessKeyID, o.queueSecretAccessKey = accessKeyID, secretAccessKey
	tb.Cleanup(func() {
		o.queueURL, o.queueRegion = prev.queueURL, prev.queueRegion
		o.queueAccessKeyID, o.queueSecretAccessKey = prev.queueAccessKeyID, prev.queueSecretAccessKey
	})
}

// SetConsumerQueue は、テスト用に consume 対象キューの接続設定をまとめて行います。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *ConsumerQueueConfig) SetConsumerQueue(
	tb testing.TB, url, dlqURL, region, accessKeyID, secretAccessKey string,
) {
	tb.Helper()
	prev := *c
	c.url, c.dlqURL, c.region = url, dlqURL, region
	c.accessKeyID, c.secretAccessKey = accessKeyID, secretAccessKey
	tb.Cleanup(func() {
		c.url, c.dlqURL, c.region = prev.url, prev.dlqURL, prev.region
		c.accessKeyID, c.secretAccessKey = prev.accessKeyID, prev.secretAccessKey
	})
}

// SetSameSite は、テスト用にセキュアクッキーの SameSite 強制値を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (s *SecureCookieConfig) SetSameSite(tb testing.TB, sameSite string) {
	tb.Helper()
	prev := s.sameSite
	s.sameSite = sameSite
	tb.Cleanup(func() { s.sameSite = prev })
}

// SetDomain は、テスト用にセキュアクッキーの Domain 強制値を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (s *SecureCookieConfig) SetDomain(tb testing.TB, domain string) {
	tb.Helper()
	prev := s.domain
	s.domain = domain
	tb.Cleanup(func() { s.domain = prev })
}

// SetAuthIssuer は、テスト用に認証の issuer を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *AuthConfig) SetAuthIssuer(tb testing.TB, issuer string) {
	tb.Helper()
	prev := a.issuer
	a.issuer = issuer
	tb.Cleanup(func() { a.issuer = prev })
}

// SetAuthAudience は、テスト用に認証の audience を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *AuthConfig) SetAuthAudience(tb testing.TB, audience string) {
	tb.Helper()
	prev := a.audience
	a.audience = audience
	tb.Cleanup(func() { a.audience = prev })
}

// SetAuthJWKSURL は、テスト用に認証の JWKS URL を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *AuthConfig) SetAuthJWKSURL(tb testing.TB, jwksURL string) {
	tb.Helper()
	prev := a.jwksURL
	a.jwksURL = jwksURL
	tb.Cleanup(func() { a.jwksURL = prev })
}
