package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/di"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingStop は、呼び出し有無と渡された停止用 context の期限を記録するフェイクの停止関数です。
type recordingStop struct {
	called      bool
	hasDeadline bool
	deadline    time.Time
}

// makeStart は、固定の done チャネルを返すフェイクの開始関数を生成します。
func makeStart(done <-chan error) di.StartFunc {
	return func(_ context.Context, _ string, _ []string) <-chan error {
		return done
	}
}

func (r *recordingStop) fn() di.StopFunc {
	return func(c context.Context) error {
		r.called = true
		r.deadline, r.hasDeadline = c.Deadline()
		return nil
	}
}

func TestRunJob(t *testing.T) {
	t.Parallel()

	t.Run("正常系_タイムアウト未指定でジョブ完了時はその結果を返し停止する", func(t *testing.T) {
		t.Parallel()

		done := make(chan error, 1)
		done <- nil
		stop := &recordingStop{}

		err := runJob(context.Background(), "j", nil, 0, makeStart(done), stop.fn())

		require.NoError(t, err)
		assert.True(t, stop.called, "停止処理が呼ばれること")
	})

	t.Run("異常系_タイムアウト未指定でジョブがエラーを返すとそのエラーを返す", func(t *testing.T) {
		t.Parallel()

		jobErr := errors.New("job failed")
		done := make(chan error, 1)
		done <- jobErr
		stop := &recordingStop{}

		err := runJob(context.Background(), "j", nil, 0, makeStart(done), stop.fn())

		require.ErrorIs(t, err, jobErr)
		assert.True(t, stop.called)
	})

	t.Run("正常系_タイムアウト指定でも期限内に完了すればジョブ結果を返す", func(t *testing.T) {
		t.Parallel()

		done := make(chan error, 1)
		done <- nil
		stop := &recordingStop{}

		err := runJob(context.Background(), "j", nil, 10*time.Second, makeStart(done), stop.fn())

		require.NoError(t, err)
		assert.True(t, stop.called)
	})

	t.Run("異常系_タイムアウト発火時はDeadlineExceededを返し停止に新しい猶予を与える", func(t *testing.T) {
		t.Parallel()

		// 完了通知が来ない done を渡し、短いタイムアウトを発火させる。
		done := make(chan error) // 送信されない
		stop := &recordingStop{}

		err := runJob(context.Background(), "j", nil, 20*time.Millisecond, makeStart(done), stop.fn())

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.True(t, stop.called)
		// 停止用 context は期限切れの waitCtx ではなく、停止開始時点から作り直された猶予を持つこと
		// （4a10247 の回帰防止）。残り時間が十分大きいことで「新しい猶予」を確認する。
		require.True(t, stop.hasDeadline)
		assert.Greater(t, time.Until(stop.deadline), 20*time.Second)
	})

	t.Run("異常系_親contextキャンセル時はそのエラーを返し停止処理を流す", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error) // 送信されない
		stop := &recordingStop{}

		cancel() // 親をキャンセル済みにしてから実行する

		err := runJob(ctx, "j", nil, 10*time.Second, makeStart(done), stop.fn())

		require.ErrorIs(t, err, context.Canceled)
		assert.True(t, stop.called)
	})
}

func TestRunJobWith(t *testing.T) {
	t.Parallel()

	t.Run("正常系_provideで取得したstart/stopをrunJobへ渡し結果を返す", func(t *testing.T) {
		t.Parallel()

		done := make(chan error, 1)
		done <- nil
		stop := &recordingStop{}

		provide := func() (di.StartFunc, di.StopFunc) {
			return makeStart(done), stop.fn()
		}

		err := RunJobWith(context.Background(), "j", nil, 0, provide)

		require.NoError(t, err)
		assert.True(t, stop.called, "provide 由来の停止処理が呼ばれること")
	})
}
