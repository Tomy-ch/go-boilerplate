package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

			err := runJob(context.Background(), "j", nil, 0, makeStart(done), stop.fn())

			require.NoError(t, err)
			assert.True(t, stop.called, "停止処理が呼ばれること")
		})

		t.Run("タイムアウト指定でも期限内に完了すればジョブ結果を返す", func(t *testing.T) {
			t.Parallel()

			done := make(chan error, 1)
			done <- nil
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 10*time.Second, makeStart(done), stop.fn())

			require.NoError(t, err)
			assert.True(t, stop.called)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("タイムアウト未指定でジョブがエラーを返すとそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			jobErr := errors.New("job failed")
			done := make(chan error, 1)
			done <- jobErr
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 0, makeStart(done), stop.fn())

			require.ErrorIs(t, err, jobErr)
			assert.True(t, stop.called)
		})

		t.Run("タイムアウト指定でも期限内にジョブがエラー完了すればそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			// timeout > 0 だが期限到達前に done<-jobErr が届くケース（select の done arm 経路）。
			jobErr := errors.New("job failed")
			done := make(chan error, 1)
			done <- jobErr
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 10*time.Second, makeStart(done), stop.fn())

			require.ErrorIs(t, err, jobErr)
			assert.True(t, stop.called)
		})

		t.Run("タイムアウト発火時はDeadlineExceededを返し停止に新しい猶予を与える", func(t *testing.T) {
			t.Parallel()

			// 完了通知が来ない done を渡し、短いタイムアウトを発火させる。
			done := make(chan error) // 送信されない
			stop := &recordingStop{}

			err := runJob(context.Background(), "j", nil, 20*time.Millisecond, makeStart(done), stop.fn())

			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.True(t, stop.called)
			// 停止用 context は期限切れの waitCtx ではなく、停止開始時点から作り直された猶予を持つこと
			// （4a10247 の回帰防止）。stopTimeout 定数を基準に「ライブ時刻ではなく定数 ÷ 2 を超える猶予」
			// として固定し、 stopTimeout の値変更にも追随する。
			require.True(t, stop.hasDeadline)
			assert.Greater(t, time.Until(stop.deadline), stopTimeout/2)
		})

		t.Run("親contextキャンセル時はそのエラーを返し停止処理を流す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error) // 送信されない
			stop := &recordingStop{}

			cancel() // 親をキャンセル済みにしてから実行する

			err := runJob(ctx, "j", nil, 10*time.Second, makeStart(done), stop.fn())

			require.ErrorIs(t, err, context.Canceled)
			assert.True(t, stop.called)
		})

		t.Run("本体成功でも停止が失敗すれば停止エラーを返す", func(t *testing.T) {
			t.Parallel()

			// 停止失敗（OTel flush / DB pool close 等）が exit code へ反映されるよう
			// 本体結果(nil)と Join して非 nil を返すこと。
			stopErr := errors.New("stop failed")
			done := make(chan error, 1)
			done <- nil
			stop := &recordingStop{err: stopErr}

			err := runJob(context.Background(), "j", nil, 0, makeStart(done), stop.fn())

			require.ErrorIs(t, err, stopErr)
		})

		t.Run("タイムアウト発火かつ停止失敗時は両方のエラーが取れる", func(t *testing.T) {
			t.Parallel()

			// 本体側(DeadlineExceeded)と停止側(stopErr)の双方が errors.Is で取得できること。
			stopErr := errors.New("stop failed")
			done := make(chan error) // 送信されない
			stop := &recordingStop{err: stopErr}

			err := runJob(context.Background(), "j", nil, 20*time.Millisecond, makeStart(done), stop.fn())

			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.ErrorIs(t, err, stopErr)
		})
	})
}

func TestGracefulStop(t *testing.T) {
	t.Parallel()

	t.Run("親ctxがキャンセル済みでも停止用ctxは全猶予を持ち期限切れでない", func(t *testing.T) {
		t.Parallel()

		// 親 ctx を事前にキャンセルしておく（SIGINT 伝播・waitCtx タイムアウト後の状況を模倣）。
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		stop := &recordingStop{}

		_ = gracefulStop(ctx, stop.fn())

		// 停止処理は呼ばれ、渡された context は期限切れでなく stopTimeout 相当の猶予を持つこと。
		require.True(t, stop.called, "停止処理が呼ばれること")
		require.True(t, stop.hasDeadline, "停止用 context に deadline があること")
		require.Nil(t, stop.ctxErr, "停止用 context が期限切れでないこと")
		assert.Greater(t, time.Until(stop.deadline), stopTimeout/2, "stopTimeout 相当の猶予が残っていること")
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

			err := RunJobWith(context.Background(), "j", nil, 0, provide)

			require.NoError(t, err)
			assert.True(t, stop.called, "provide 由来の停止処理が呼ばれること")
		})
	})
}
