package dbslot

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PgxAdmin は実 Postgres への pgx アダプタ。共有 DB（localhost:5432）へ実接続して検証する統合テスト。
func testAdmin() *PgxAdmin {
	return NewPgxAdmin("localhost", 5432, "postgres", "postgres-password", "postgres")
}

func dropTestDB(t *testing.T, a *PgxAdmin, name string) {
	t.Helper()
	ctx := context.Background()
	conn, err := a.connect(ctx, a.maintenanceDB)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close(ctx) }()
	_, _ = conn.Exec(ctx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", name)
	_, _ = conn.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize())
}

func TestPgxAdmin_connect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("maintenance DB へ接続できる", func(t *testing.T) {
			t.Parallel()

			a := testAdmin()
			conn, err := a.connect(context.Background(), a.maintenanceDB)
			require.NoError(t, err)
			_ = conn.Close(context.Background())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("到達不能なホストではエラーを返す", func(t *testing.T) {
			t.Parallel()

			a := NewPgxAdmin("localhost", 1, "postgres", "x", "postgres")
			_, err := a.connect(context.Background(), a.maintenanceDB)
			require.Error(t, err)
		})
	})
}

//nolint:paralleltest // 実 DB 状態を作って検証するため並列化不可
func TestPgxAdmin_EnsureDatabase(t *testing.T) {
	a := testAdmin()
	ctx := context.Background()
	const name = "dbslot_it_ensure"
	dropTestDB(t, a, name)
	t.Cleanup(func() { dropTestDB(t, a, name) })

	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
		t.Run("未作成なら作成し、既存なら冪等に成功する", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
			require.NoError(t, a.EnsureDatabase(ctx, name))
			require.NoError(t, a.EnsureDatabase(ctx, name))

			conn, err := a.connect(ctx, a.maintenanceDB)
			require.NoError(t, err)
			defer func() { _ = conn.Close(ctx) }()
			var exists bool
			require.NoError(t, conn.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)", name).Scan(&exists))
			assert.True(t, exists)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
		t.Run("接続不能ならエラーを返す", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
			bad := NewPgxAdmin("localhost", 1, "postgres", "x", "postgres")
			require.Error(t, bad.EnsureDatabase(ctx, name))
		})
	})
}

//nolint:paralleltest // 実 DB 状態を作って検証するため並列化不可
func TestPgxAdmin_SetupDatabase(t *testing.T) {
	a := testAdmin()
	ctx := context.Background()
	const name = "dbslot_it_setup"
	dropTestDB(t, a, name)
	t.Cleanup(func() { dropTestDB(t, a, name) })
	require.NoError(t, a.EnsureDatabase(ctx, name))

	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
		t.Run("timezone と pg_trgm 拡張を設定する", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
			require.NoError(t, a.SetupDatabase(ctx, name))

			conn, err := a.connect(ctx, name)
			require.NoError(t, err)
			defer func() { _ = conn.Close(ctx) }()

			var tz string
			require.NoError(t, conn.QueryRow(ctx, "SHOW timezone").Scan(&tz))
			assert.Equal(t, "Asia/Tokyo", tz)

			var hasExt bool
			require.NoError(t, conn.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='pg_trgm')").Scan(&hasExt))
			assert.True(t, hasExt)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
		t.Run("接続不能ならエラーを返す", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
			bad := NewPgxAdmin("localhost", 1, "postgres", "x", "postgres")
			require.Error(t, bad.SetupDatabase(ctx, name))
		})
	})
}

//nolint:paralleltest // 実 DB 状態を作って検証するため並列化不可
func TestPgxAdmin_ActiveConnections(t *testing.T) {
	a := testAdmin()
	ctx := context.Background()
	const name = "dbslot_it_conns"
	dropTestDB(t, a, name)
	t.Cleanup(func() { dropTestDB(t, a, name) })
	require.NoError(t, a.EnsureDatabase(ctx, name))

	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
		t.Run("名前未指定は 0 を返す", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
			n, err := a.ActiveConnections(ctx)
			require.NoError(t, err)
			assert.Equal(t, 0, n)
		})

		t.Run("接続が無ければ 0、繋げば 1 以上を返す", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
			n0, err := a.ActiveConnections(ctx, name)
			require.NoError(t, err)
			assert.Equal(t, 0, n0)

			conn, err := a.connect(ctx, name)
			require.NoError(t, err)
			defer func() { _ = conn.Close(ctx) }()
			require.NoError(t, conn.Ping(ctx))

			n1, err := a.ActiveConnections(ctx, name)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, n1, 1)
		})
	})

	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
		t.Run("接続不能ならエラーを返す", func(t *testing.T) { //nolint:paralleltest // 実 DB 依存
			bad := NewPgxAdmin("localhost", 1, "postgres", "x", "postgres")
			_, err := bad.ActiveConnections(ctx, name)
			require.Error(t, err)
		})
	})
}

func TestNewPgxAdmin(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPgxAdmin_dsn(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
