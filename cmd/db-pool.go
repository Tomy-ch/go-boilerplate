package main

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go-boilerplate/internal/cli/dbpool"

	"github.com/spf13/cobra"
)

const (
	defaultPoolMaxSlots     = 8
	defaultPoolTTLSeconds   = 1800
	defaultPoolAPIBasePort  = 8080
	defaultPoolMockBasePort = 4000
	defaultPoolPGPort       = 5432
)

// newDBPoolCommand は、worktree 並列開発用の DB スロットプールを操作するコマンドを生成します。
// 詳細は docs/maintenance/db-worktree-pool.md。
func newDBPoolCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db-pool",
		Short: "worktree 並列開発用の DB スロットプールを操作します（acquire/release/heartbeat/status）。",
	}
	cmd.AddCommand(
		newDBPoolSubCommand("acquire", "空きスロットをリースし共有 DB に自 worktree の DB を用意します。",
			func(ctx context.Context, p *dbpool.Pool) error { return p.Acquire(ctx) }),
		newDBPoolSubCommand("release", "保持中のスロットを解放します（serve コンテナ停止・DB は warm 保持）。",
			func(ctx context.Context, p *dbpool.Pool) error { return p.Release(ctx) }),
		newDBPoolSubCommand("heartbeat", "保持中スロットの heartbeat を更新します。",
			func(_ context.Context, p *dbpool.Pool) error { return p.Heartbeat() }),
		newDBPoolSubCommand("status", "スロットの占有状況を表示します。",
			func(_ context.Context, p *dbpool.Pool) error { return p.Status() }),
	)
	return cmd
}

func newDBPoolSubCommand(use, short string, run func(context.Context, *dbpool.Pool) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(c *cobra.Command, _ []string) error {
			pool, err := newPool()
			if err != nil {
				return err
			}
			return run(c.Context(), pool)
		},
	}
}

// newPool は、実依存（ホスト実行の registry / pgx admin / docker compose）を配線して Pool を生成します。
func newPool() (*dbpool.Pool, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	reg := dbpool.NewRegistry(
		poolDir(),
		root,
		gitBranch(),
		time.Duration(envInt("GOBP_DB_POOL_TTL", defaultPoolTTLSeconds))*time.Second,
		envInt("GOBP_DB_POOL_MAX", defaultPoolMaxSlots),
		nil,
	)
	admin := dbpool.NewPgxAdmin(
		envStr("GOBP_DB_POOL_PGHOST", "localhost"),
		envInt("GOBP_DB_POOL_PGPORT", defaultPoolPGPort),
		envStr("GOBP_DB_POOL_PGUSER", "postgres"),
		envStr("GOBP_DB_POOL_PGPASSWORD", "postgres-password"),
		envStr("GOBP_DB_POOL_PGMAINTDB", "postgres"),
	)
	cfg := dbpool.Config{
		Root:          root,
		SharedProject: envStr("GOBP_DB_SHARED_PROJECT", "gobp-shared"),
		APIBasePort:   envInt("GOBP_API_POOL_BASE", defaultPoolAPIBasePort),
		MockAuthBase:  envInt("GOBP_MOCK_AUTH_POOL_BASE", defaultPoolMockBasePort),
		APPEnv:        os.Getenv("APP_ENV"),
	}
	return dbpool.NewPool(reg, admin, dbpool.ExecCompose{}, cfg, os.Stdout, os.Stderr), nil
}

func poolDir() string {
	if v := os.Getenv("GOBP_DB_POOL_DIR"); v != "" {
		return v
	}
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		home, _ := os.UserHomeDir()
		cache = home + "/.cache"
	}
	return cache + "/gobp-db-pool"
}

func gitBranch() string {
	out, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "-"
	}
	return strings.TrimSpace(string(out))
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
