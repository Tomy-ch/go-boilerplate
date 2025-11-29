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
func (a *ApplicationConfig) SetServerAppMode(t testing.TB, mode string) {
	t.Helper()
	prev := a.AppMode()
	a.mode = mode
	t.Cleanup(func() { a.mode = prev })
}

// SetDatabaseDriver は、テスト用にデータベースのドライバーを設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (d *DatabaseConfig) SetDatabaseDriver(t testing.TB, driver string) {
	t.Helper()
	prev := d.DatabaseDriver()
	d.driver = driver
	t.Cleanup(func() { d.driver = prev })
}

// SetDatabaseHost は、テスト用にデータベースのホスト名を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (d *DatabaseConfig) SetDatabaseHost(t testing.TB, host string) {
	t.Helper()
	prev := d.DatabaseHost()
	d.host = host
	t.Cleanup(func() { d.host = prev })
}

// SetDatabaseName は、テスト用にデータベースの名前を設定します。
//
// 実行後は、元の値に戻すためのクリーンアップ関数が登録されます。
func (d *DatabaseConfig) SetDatabaseName(t testing.TB, name string) {
	t.Helper()
	prev := d.DatabaseName()
	d.name = name
	t.Cleanup(func() { d.name = prev })
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
