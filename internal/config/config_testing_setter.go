package config

import (
	"net"
	"testing"
)

// WARN: 本番コードでは使用しないでください。テスト用の設定を行うためのメソッドです。
//
// 実装メソッドは無闇に増やさず、必要なものだけを追加してください。

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

// SetHeaderName は、テスト用に認証のヘッダ名を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *AuthConfig) SetHeaderName(tb testing.TB, headerName string) {
	tb.Helper()
	prev := a.headerName
	a.headerName = headerName
	tb.Cleanup(func() { a.headerName = prev })
}

// SetAllowedHeaderBearer は、テスト用に認証の許可されたヘッダベアラーを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *AuthConfig) SetAllowedHeaderBearer(tb testing.TB, allowed bool) {
	tb.Helper()
	prev := a.allowedHeaderBearer
	a.allowedHeaderBearer = allowed
	tb.Cleanup(func() { a.allowedHeaderBearer = prev })
}
