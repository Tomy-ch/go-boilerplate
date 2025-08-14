package config

import (
	"net"
	"testing"
	"time"
)

// WARN: 本番コードでは使用しないでください。テスト用の設定を行うためのメソッドです。
//
// 実装メソッドは無闇に増やさず、必要なものだけを追加してください。

// SetServerAppMode は、テスト用にサーバーのAppModeを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetServerAppMode(t testing.TB, mode string) {
	t.Helper()
	prev := c.ServerAppMode()
	c.server.appMode = mode
	t.Cleanup(func() { c.server.appMode = prev })
}

// SetDatabaseHost は、テスト用にデータベースのホスト名を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDatabaseHost(t testing.TB, host string) {
	t.Helper()
	prev := c.DatabaseHost()
	c.database.host = host
	t.Cleanup(func() { c.database.host = prev })
}

// SetDatabasePort は、テスト用にデータベースのポート番号を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDatabasePort(t testing.TB, port int) {
	t.Helper()
	prev := c.DatabasePort()
	c.database.port = port
	t.Cleanup(func() { c.database.port = prev })
}

// SetDatabaseUser は、テスト用にデータベースのユーザー名を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDatabaseUser(t testing.TB, user string) {
	t.Helper()
	prev := c.DatabaseUser()
	c.database.user = user
	t.Cleanup(func() { c.database.user = prev })
}

// SetDatabasePassword は、テスト用にデータベースのパスワードを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDatabasePassword(t testing.TB, pass string) {
	t.Helper()
	prev := c.DatabasePassword()
	c.database.password = pass
	t.Cleanup(func() { c.database.password = prev })
}

// SetDatabaseName は、テスト用にデータベースの名前を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDatabaseName(t testing.TB, name string) {
	t.Helper()
	prev := c.DatabaseName()
	c.database.name = name
	t.Cleanup(func() { c.database.name = prev })
}

// SetDatabaseSSLMode は、テスト用にデータベースのSSLモードを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDatabaseSSLMode(t testing.TB, mode string) {
	t.Helper()
	prev := c.DatabaseSSLMode()
	c.database.sslMode = mode
	t.Cleanup(func() { c.database.sslMode = prev })
}

// SetDBMaxOpenConns は、テスト用にデータベースの最大オープン接続数を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDBMaxOpenConns(t testing.TB, maxVal int) {
	t.Helper()
	prev := c.DBMaxOpenConns()
	c.database.connection.maxOpenConns = maxVal
	t.Cleanup(func() { c.database.connection.maxOpenConns = prev })
}

// SetDBMaxIdleConns は、テスト用にデータベースの最大アイドル接続数を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDBMaxIdleConns(t testing.TB, maxVal int) {
	t.Helper()
	prev := c.DBMaxIdleConns()
	c.database.connection.maxIdleConns = maxVal
	t.Cleanup(func() { c.database.connection.maxIdleConns = prev })
}

// SetDBConnMaxLifetime は、テスト用にデータベースの接続の最大寿命を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDBConnMaxLifetime(t testing.TB, d time.Duration) {
	t.Helper()
	prev := c.DBConnMaxLifetime()
	c.database.connection.maxLifetime = d
	t.Cleanup(func() { c.database.connection.maxLifetime = prev })
}

// SetCIDR は、テスト用にセキュリティのCIDRを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetCIDR(t testing.TB, cidr *net.IPNet) {
	t.Helper()
	prev := c.CIDR()
	c.security.cidr = cidr
	t.Cleanup(func() { c.security.cidr = prev })
}
