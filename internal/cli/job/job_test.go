package job

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testGrace は、テスト用の停止猶予（APP_SHUTDOWN_TIMEOUT 相当）です。
const testGrace = 30 * time.Second

// recordingStop は、呼び出し有無・停止用 context の期限・ctx のエラー状態を記録するフェイクの停止関数です。
type recordingStop struct {
	called      bool
	hasDeadline bool
	deadline    time.Time
	ctxErr      error // fn 呼び出し時点の停止用 context のエラー状態
	err         error // fn が返す停止エラー（nil なら成功）
}

// makeStart は、固定の done チャネルを返すフェイクの開始関数を生成します。
func makeStart(done <-chan error) StartFunc {
	return func(_ context.Context, _ string, _ []string) <-chan error {
		return done
	}
}

func (r *recordingStop) fn() StopFunc {
	return func(c context.Context) error {
		r.called = true
		r.deadline, r.hasDeadline = c.Deadline()
		r.ctxErr = c.Err()
		return r.err
	}
}

func TestRunJob(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("タイムアウト未指定でジョブ完了時はその結果を返し停止する", func(t *testing.T) {
			t.Parallel()

			done := make(chan error, 1)
			done <- nil
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 0, testGrace, makeStart(done), stop.fn())

			require.NoError(t, err)
			assert.True(t, stop.called, "停止処理が呼ばれること")
		})

		t.Run("タイムアウト指定でも期限内に完了すればジョブ結果を返す", func(t *testing.T) {
			t.Parallel()

			done := make(chan error, 1)
			done <- nil
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 10*time.Second, testGrace, makeStart(done), stop.fn())

			require.NoError(t, err)
			assert.True(t, stop.called)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("タイムアウト未指定でジョブがエラーを返すとそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			jobErr := xerrors.New("job failed")
			done := make(chan error, 1)
			done <- jobErr
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 0, testGrace, makeStart(done), stop.fn())

			require.ErrorIs(t, err, jobErr)
			assert.True(t, stop.called)
		})

		t.Run("タイムアウト指定でも期限内にジョブがエラー完了すればそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			jobErr := xerrors.New("job failed")
			done := make(chan error, 1)
			done <- jobErr
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 10*time.Second, testGrace, makeStart(done), stop.fn())

			require.ErrorIs(t, err, jobErr)
			assert.True(t, stop.called)
		})

		t.Run("タイムアウト発火時はDeadlineExceededを返し停止に新しい猶予を与える", func(t *testing.T) {
			t.Parallel()

			done := make(chan error) // 送信されない
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 20*time.Millisecond, testGrace, makeStart(done), stop.fn())

			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.True(t, stop.called)
			// 停止用 context は期限切れの waitCtx ではなく、停止開始時点から作り直された猶予（grace）を持つこと。
			// grace を基準に「ライブ時刻ではなく grace ÷ 2 を超える猶予」として固定し、grace の値変更にも追随する。
			require.True(t, stop.hasDeadline)
			assert.Greater(t, time.Until(stop.deadline), testGrace/2)
		})

		t.Run("親contextキャンセル時はそのエラーを返し停止処理を流す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error) // 送信されない
			stop := &recordingStop{}

			cancel() // 親をキャンセル済みにしてから実行する

			err := runJob(ctx, "j", nil, 10*time.Second, testGrace, makeStart(done), stop.fn())

			require.ErrorIs(t, err, context.Canceled)
			assert.True(t, stop.called)
		})

		t.Run("本体成功でも停止が失敗すれば停止エラーを返す", func(t *testing.T) {
			t.Parallel()

			stopErr := xerrors.New("stop failed")
			done := make(chan error, 1)
			done <- nil
			stop := &recordingStop{err: stopErr}

			err := runJob(context.Background(), "j", nil, 0, testGrace, makeStart(done), stop.fn())

			require.ErrorIs(t, err, stopErr)
		})

		t.Run("タイムアウト発火かつ停止失敗時は両方のエラーが取れる", func(t *testing.T) {
			t.Parallel()

			stopErr := xerrors.New("stop failed")
			done := make(chan error) // 送信されない
			stop := &recordingStop{err: stopErr}

			err := runJob(context.Background(), "j", nil, 20*time.Millisecond, testGrace, makeStart(done), stop.fn())

			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.ErrorIs(t, err, stopErr)
		})
	})
}

func TestGracefulStop(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("親ctxがキャンセル済みでも停止用ctxは全猶予を持ち期限切れでない", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			stop := &recordingStop{}

			_ = gracefulStop(ctx, testGrace, stop.fn())

			assert.True(t, stop.called, "停止処理が呼ばれること")
			require.True(t, stop.hasDeadline, "停止用 context に deadline があること")
			require.NoError(t, stop.ctxErr, "停止用 context が期限切れでないこと")
			assert.Greater(t, time.Until(stop.deadline), testGrace/2, "grace 相当の猶予が残っていること")
		})
	})
}

func TestRunJobWith(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("provideで取得したstart/stopをrunJobへ渡し結果を返す", func(t *testing.T) {
			t.Parallel()

			done := make(chan error, 1)
			done <- nil
			stop := &recordingStop{}

			provide := func() (StartFunc, StopFunc) {
				return makeStart(done), stop.fn()
			}

			err := RunJobWith(context.Background(), "j", nil, 0, testGrace, provide)

			require.NoError(t, err)
			assert.True(t, stop.called, "provide 由来の停止処理が呼ばれること")
		})
	})
}
