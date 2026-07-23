package dbslot

import "testing"

// PgxAdmin は実 Postgres への薄い pgx アダプタで、分岐は接続失敗/存在チェックの委譲のみ。
// 純粋な単体テストでは実 DB を要するため、これらは Pool レベルの統合フローでカバーする方針とし、ここでは 1:1 規約に従い Skip を明示する。

func TestPgxAdmin_connect(t *testing.T) {
	t.Parallel()
	t.Skip("実 DB 依存の薄い pgx アダプタのため統合テストで検証")
}

func TestPgxAdmin_EnsureDatabase(t *testing.T) {
	t.Parallel()
	t.Skip("実 DB 依存の薄い pgx アダプタのため統合テストで検証")
}

func TestPgxAdmin_SetupDatabase(t *testing.T) {
	t.Parallel()
	t.Skip("実 DB 依存の薄い pgx アダプタのため統合テストで検証")
}

func TestPgxAdmin_ActiveConnections(t *testing.T) {
	t.Parallel()
	t.Skip("実 DB 依存の薄い pgx アダプタのため統合テストで検証")
}
