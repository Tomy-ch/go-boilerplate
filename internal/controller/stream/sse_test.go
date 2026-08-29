package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/controller/stream/gen"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// sseTestDeadline は、encoder のテストで 1 回の書き込みに与える猶予です。
const sseTestDeadline = 2 * time.Second

// sseTestOccurredAt は、封筒の発生時刻として使う基準時刻です。
var sseTestOccurredAt = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// sseTestEvent は、payload を持つ business event を組み立てます。封筒の骨は testEvent と共有します。
func sseTestEvent(seq rt.Sequence) rt.DeliveryEvent {
	e := testEvent(seq)
	e.Payload = json.RawMessage(`{"k":"v"}`)

	return e
}

// captureSSE は、実 network 上で body を走らせ、client が受け取った生のバイト列を返します。
// httptest.ResponseRecorder では write deadline も flush も検証できないため、実サーバーを立てます。
func captureSSE(t *testing.T, body func(w *sseWriter) error) (string, error) {
	t.Helper()

	var writeErr error
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
		defer close(done)
		writeErr = body(newSSEWriter(res, sseTestDeadline))
	}))
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })

	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	<-done

	return string(raw), writeErr
}

func Test_newSSEWriter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した ResponseWriter へ書ける writer を返す", func(t *testing.T) {
			t.Parallel()

			got, err := captureSSE(t, func(w *sseWriter) error { return w.writeHeartbeat() })

			require.NoError(t, err)
			assert.Equal(t, ": heartbeat\n\n", got)
		})
	})
}

func Test_sseWriter_commit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SSE のヘッダを付けて 200 を確定する", func(t *testing.T) {
			t.Parallel()

			var commitErr error
			srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
				commitErr = newSSEWriter(res, sseTestDeadline).commit()
			}))
			t.Cleanup(srv.Close)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			res, err := srv.Client().Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = res.Body.Close() })
			_, err = io.ReadAll(res.Body)
			require.NoError(t, err)

			require.NoError(t, commitErr)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))
			assert.Equal(t, "no-store", res.Header.Get("Cache-Control"))
			assert.Equal(t, "no", res.Header.Get("X-Accel-Buffering"))
		})
	})
}

func Test_sseWriter_writeEvent(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("id に sequence を載せて封筒を書く", func(t *testing.T) {
			t.Parallel()

			got, err := captureSSE(t, func(w *sseWriter) error { return w.writeEvent(sseTestEvent(42)) })

			require.NoError(t, err)
			lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
			require.Len(t, lines, 2, "business event は id 行と data 行の 2 行で 1 フレーム")
			assert.Equal(t, "id: 42", lines[0])
			// フレーミングと各フィールドは wire 契約だが、JSON のフィールド順は MarshalJSON の実装詳細。
			assert.JSONEq(t,
				`{"eventId":"evt-42","streamId":"stream-1","sequence":"42",`+
					`"type":"sample.thing.created.v1","occurredAt":"2026-01-01T00:00:00Z",`+
					`"schemaVersion":1,"payload":{"k":"v"}}`,
				strings.TrimPrefix(lines[1], "data: "))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("直列化できない封筒はエラーを返し何も書かない", func(t *testing.T) {
			t.Parallel()

			broken := sseTestEvent(1)
			broken.Payload = json.RawMessage(`{`)

			got, err := captureSSE(t, func(w *sseWriter) error { return w.writeEvent(broken) })

			require.ErrorIs(t, err, rt.ErrInvalidEvent)
			assert.Empty(t, got)
		})
	})
}

func Test_sseWriter_writeControl(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("event: control を id 無しで書く", func(t *testing.T) {
			t.Parallel()

			got, err := captureSSE(t, func(w *sseWriter) error {
				return w.writeControl(gen.ControlEvent{Action: gen.RECONNECT, Reason: gen.SERVERDRAINING})
			})

			require.NoError(t, err)
			assert.Equal(t, "event: control\ndata: {\"action\":\"RECONNECT\",\"reason\":\"SERVER_DRAINING\"}\n\n", got)
			assert.NotContains(t, got, "id:", "control event が Last-Event-ID を汚さないこと")
		})

		t.Run("retryAfterMs を持つ指示はそれも載せる", func(t *testing.T) {
			t.Parallel()

			ms := 5000
			got, err := captureSSE(t, func(w *sseWriter) error {
				return w.writeControl(gen.ControlEvent{
					Action: gen.RETRYLATER, Reason: gen.TEMPORARILYOVERLOADED, RetryAfterMs: &ms,
				})
			})

			require.NoError(t, err)
			assert.Contains(t, got, `"retryAfterMs":5000`)
		})
	})
}

func Test_sseWriter_writeHeartbeat(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("id を持たない comment を書く", func(t *testing.T) {
			t.Parallel()

			got, err := captureSSE(t, func(w *sseWriter) error { return w.writeHeartbeat() })

			require.NoError(t, err)
			assert.Equal(t, ": heartbeat\n\n", got)
			assert.NotContains(t, got, "id:", "heartbeat が Last-Event-ID を汚さないこと")
		})
	})
}

func Test_sseWriter_write(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("連続して書いても deadline を張り直すので切れない", func(t *testing.T) {
			t.Parallel()

			got, err := captureSSE(t, func(w *sseWriter) error {
				if err := w.writeEvent(sseTestEvent(1)); err != nil {
					return err
				}

				return w.writeEvent(sseTestEvent(2))
			})

			require.NoError(t, err)
			assert.Contains(t, got, "id: 1\n")
			assert.Contains(t, got, "id: 2\n")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("deadline を張れない writer ではエラーを返す", func(t *testing.T) {
			t.Parallel()

			// ResponseRecorder は SetWriteDeadline を持たないため、書き込み前に失敗する。
			err := newSSEWriter(httptest.NewRecorder(), sseTestDeadline).writeHeartbeat()

			require.Error(t, err)
		})
	})
}

func Test_sseWriter_flush(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("レスポンスが終わる前に client が読み取れる", func(t *testing.T) {
			t.Parallel()

			release := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
				w := newSSEWriter(res, sseTestDeadline)
				if err := w.commit(); err != nil {
					return
				}
				_ = w.writeEvent(sseTestEvent(1))
				<-release
			}))
			t.Cleanup(func() { close(release); srv.Close() })

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			res, err := srv.Client().Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = res.Body.Close() })

			line, err := bufio.NewReader(res.Body).ReadString('\n')

			require.NoError(t, err)
			assert.Equal(t, "id: 1\n", line)
		})
	})
}
