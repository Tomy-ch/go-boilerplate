package config

import (
	"net"
	"testing"
	"time"
)

// WARN: 本番コードでは使用しないでください。テスト用の設定を行うためのメソッドです。
//
// 実装メソッドは無闇に増やさず、必要なものだけを追加してください。

// SetApplicationMode は、テスト用にサーバーのAppModeを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *ApplicationConfig) SetApplicationMode(t testing.TB, mode string) {
	t.Helper()
	prev := a.Mode()
	a.mode = mode
	t.Cleanup(func() { a.mode = prev })
}

// SetApplicationEnv は、テスト用にアプリケーションの環境を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *ApplicationConfig) SetApplicationEnv(t testing.TB, env string) {
	t.Helper()
	prev := a.Env()
	a.env = env
	t.Cleanup(func() { a.env = prev })
}

// SetServerPort は、テスト用にサーバーのポートを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (s *ServerConfig) SetServerPort(t testing.TB, port int) {
	t.Helper()
	prev := s.Port()
	s.port = port
	t.Cleanup(func() { s.port = prev })
}

// SetObservabilityMaskedDBQueryArgs は、テスト用にオブザーバビリティのDBクエリ引数の設定を行います。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (o *ObservabilityConfig) SetObservabilityMaskedDBQueryArgs(t testing.TB, val bool) {
	t.Helper()
	prev := o.MaskedDBQueryArgs()
	o.maskedDBQueryArgs = val
	t.Cleanup(func() { o.maskedDBQueryArgs = prev })
}

// SetDatabaseHost は、テスト用にデータベースのホスト名を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (d *DatabaseConfig) SetDatabaseHost(t testing.TB, host string) {
	t.Helper()
	prev := d.Host()
	d.host = host
	t.Cleanup(func() { d.host = prev })
}

// SetDatabaseName は、テスト用にデータベースの名前を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (d *DatabaseConfig) SetDatabaseName(t testing.TB, name string) {
	t.Helper()
	prev := d.DBName()
	d.name = name
	t.Cleanup(func() { d.name = prev })
}

// SetDatabasePassword は、テスト用にデータベースのパスワードを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (d *DatabaseConfig) SetDatabasePassword(t testing.TB, password string) {
	t.Helper()
	prev := d.Password()
	d.password = password
	t.Cleanup(func() { d.password = prev })
}

// SetMaxConns は、テスト用にDB接続の最大オープン数を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *DBConnectionConfig) SetMaxConns(t testing.TB, maxConns int32) {
	t.Helper()
	prev := c.MaxConns()
	c.maxOpenConns = maxConns
	t.Cleanup(func() { c.maxOpenConns = prev })
}

// SetCIDR は、テスト用にセキュリティのCIDRを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (s *SecurityConfig) SetCIDR(t testing.TB, cidr *net.IPNet) {
	t.Helper()
	prev := s.CIDR()
	s.cidr = cidr
	t.Cleanup(func() { s.cidr = prev })
}

// SetCleanupInterval は、テスト用にIPレート制限のクリーンアップ間隔を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (i *IPRateLimitConfig) SetCleanupInterval(t testing.TB, interval time.Duration) {
	t.Helper()
	prev := i.cleanupInterval
	i.cleanupInterval = interval
	t.Cleanup(func() { i.cleanupInterval = prev })
}

// SetHeaderName は、テスト用に認証のヘッダ名を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *AuthConfig) SetHeaderName(t testing.TB, headerName string) {
	t.Helper()
	prev := a.headerName
	a.headerName = headerName
	t.Cleanup(func() { a.headerName = prev })
}

// SetAllowedHeaderBearer は、テスト用に認証の許可されたヘッダベアラーを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (a *AuthConfig) SetAllowedHeaderBearer(t testing.TB, allowed bool) {
	t.Helper()
	prev := a.allowedHeaderBearer
	a.allowedHeaderBearer = allowed
	t.Cleanup(func() { a.allowedHeaderBearer = prev })
}
