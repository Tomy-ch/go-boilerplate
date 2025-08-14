package config

import (
	"net"
	"testing"
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

// SetDatabaseDriver は、テスト用にデータベースのドライバーを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDatabaseDriver(t testing.TB, driver string) {
	t.Helper()
	prev := c.DatabaseDriver()
	c.database.driver = driver
	t.Cleanup(func() { c.database.driver = prev })
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

// SetDatabaseName は、テスト用にデータベースの名前を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (c *Config) SetDatabaseName(t testing.TB, name string) {
	t.Helper()
	prev := c.DatabaseName()
	c.database.name = name
	t.Cleanup(func() { c.database.name = prev })
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
