package dbpool

import (
	"context"
	"os"
	"os/exec"

	"go-boilerplate/pkg/xerrors"
)

// Compose は、docker compose 操作を抽象化します（テストでフェイク可能）。
type Compose interface {
	// UpSharedDB は、共有 DB コンテナを固定プロジェクトで起動し healthcheck 完了まで待ちます。
	UpSharedDB(ctx context.Context, project string) error
	// DownServe は、per-worktree の serve プロジェクト（app コンテナ）を停止・削除します。
	DownServe(ctx context.Context, project string) error
}

// ExecCompose は、docker compose をホストで実行する Compose 実装です。
// 出力（進捗ログ）は stderr へ流します。
type ExecCompose struct{}

// UpSharedDB は `docker compose --profile database up -d --wait database` を実行します。
func (c ExecCompose) UpSharedDB(ctx context.Context, project string) error {
	return c.run(ctx, project, "--profile", "database", "up", "-d", "--wait", "database")
}

// DownServe は `docker compose -f docker-compose.yaml -f docker-compose.pool.yaml down` を実行します。
// serve コンテナが起動していなくてもエラーにしません（冪等）。
func (c ExecCompose) DownServe(ctx context.Context, project string) error {
	_ = c.run(ctx, project, "-f", "docker-compose.yaml", "-f", "docker-compose.pool.yaml", "down")
	return nil
}

func (ExecCompose) run(ctx context.Context, project string, args ...string) error {
	// 引数は本パッケージ内の固定文字列のみ（ユーザー入力を渡さない）。
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...) //nolint:gosec // 固定引数
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME="+project)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return xerrors.Wrap(err, "docker compose failed")
	}
	return nil
}
