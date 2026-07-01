package worker

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readBody は、レスポンスボディを文字列で読み出すテストヘルパです。
func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return string(body)
}

// httpGet は、context 付きで GET リクエストを実行するテストヘルパです。
func httpGet(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return res
}

func Test_healthMux(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("healthz は常に 200 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return false }))
			defer srv.Close()

			res := httpGet(t, srv.URL+"/healthz")
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "ok", readBody(t, res))
		})

		t.Run("readyz は ready が true なら 200 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return true }))
			defer srv.Close()

			res := httpGet(t, srv.URL+"/readyz")
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "ready", readBody(t, res))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("readyz は ready が false なら 503 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return false }))
			defer srv.Close()

			res := httpGet(t, srv.URL+"/readyz")
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
			assert.Equal(t, "not ready", readBody(t, res))
		})
	})
}

func TestNewHealthServer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動から停止までのライフサイクルがpanicせず完了する", func(t *testing.T) {
			t.Parallel()

			// 固定ポートの bind 衝突を避けるため、エフェメラルポート(:0)で待ち受けさせる。
			start, stop := NewHealthServer(":0", func() bool { return true }, logging.NewTestLogger(t))
			require.NotNil(t, start)
			require.NotNil(t, stop)

			assert.NotPanics(t, func() {
				start()
				stop(context.Background())
			})
		})

		t.Run("起動していないサーバーへの停止でもpanicしない", func(t *testing.T) {
			t.Parallel()

			_, stop := NewHealthServer(":0", func() bool { return true }, logging.NewTestLogger(t))

			// 起動していないサーバーへの Shutdown はエラーにならず、panic しないこと。
			assert.NotPanics(t, func() { stop(context.Background()) })
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("bind不能なaddrはListenAndServeのエラーをログする", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			// ポート番号が範囲外の addr は bind に失敗し、ErrServerClosed 以外のエラーとしてログされる。
			start, _ := NewHealthServer("127.0.0.1:999999", func() bool { return true }, logger)
			start()

			require.Eventually(t, func() bool {
				return observed.FilterMessage("health server error").Len() == 1
			}, time.Second, 5*time.Millisecond)
		})

		t.Run("進行中リクエストがあるとShutdownの期限切れをエラーとしてログする", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			entered := make(chan struct{}, 1)
			release := make(chan struct{})

			addr := reserveTCPAddr(t)
			start, stop := NewHealthServer(addr, blockingReady(entered, release), logger)
			start()

			driveInFlight(t, "http://"+addr+"/readyz", entered, release)

			// 非アイドル接続が残る状態へ、期限切れの ctx で Shutdown を当てる。
			stop(canceledContext())
			close(release)

			assert.Positive(t, observed.FilterMessage("health server shutdown error").Len())
		})
	})
}

// reserveTCPAddr は、実際に bind 可能なポートを予約し確定した addr を返します。
func reserveTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

// blockingReady は、最初の呼び出しで entered を通知し release まで待機する ready 関数を返します。
func blockingReady(entered chan<- struct{}, release <-chan struct{}) func() bool {
	return func() bool {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		return true
	}
}

// driveInFlight は、url へ接続してハンドラを進行中（非アイドル接続）にし、進入を待ちます。
func driveInFlight(t *testing.T, url string, entered <-chan struct{}, release chan struct{}) {
	t.Helper()
	go sendUntilConnected(url)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("進行中ハンドラへ到達しなかった")
	}
}

// sendUntilConnected は、bind 完了まで接続を試み、接続できたら応答を読み切ります。
func sendUntilConnected(url string) {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return
		}
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// canceledContext は、生成直後にキャンセル済みの context を返します。
func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
