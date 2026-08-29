package stream

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	mock_ucrealtime "go-boilerplate/internal/usecase/realtime/mock"
)

// testRequest は、検証を通った接続の要求です。
var testRequest = StreamRequest{Subject: "subject-1", Destination: "stream-1", Scope: "read", Cursor: 0}

// testRegistry は、mock の Replayer と合図待ちの Sleeper を持つ registry を作ります。
func testRegistry(t *testing.T, set Settings) (*Registry, *mock_ucrealtime.MockReplayer, *fakeSleeper) {
	t.Helper()

	replayer := mock_ucrealtime.NewMockReplayer(gomock.NewController(t))
	sleeper := newFakeSleeper()
	reg := NewRegistry(replayer, sleeper, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t), set)

	return reg, replayer, sleeper
}

// callStream は、Stream を直接呼びます。確定前に断られる経路の検証に使います。
func callStream(t *testing.T, reg *Registry) (*httptest.ResponseRecorder, error) {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/streams/stream-1", nil)
	rec := httptest.NewRecorder()

	return rec, reg.Stream(e.NewContext(req, rec), testRequest)
}

// serveStream は、Stream を実 network 上で駆動し、client 側の読み手を返します。
// SSE を読み続けるため client 側に timeout は設けません。
func serveStream(t *testing.T, reg *Registry) *bufio.Reader {
	t.Helper()

	finished := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		defer close(finished)
		e := echo.New()
		//nolint:contextcheck // echo.Context が内包する request の context をそのまま運ぶため
		_ = reg.Stream(e.NewContext(r, res), testRequest)
	}))

	res := doStreamRequest(t, srv)
	t.Cleanup(func() {
		_ = res.Body.Close()
		srv.Close()
		<-finished
	})

	return bufio.NewReader(res.Body)
}

// doStreamRequest は、srv へ 1 本繋いでレスポンスを返します。
func doStreamRequest(t *testing.T, srv *httptest.Server) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	res, err := srv.Client().Do(req)
	require.NoError(t, err)

	return res
}

// readLine は、1 行読みます。届かなければテストを落とします。
func readLine(t *testing.T, r *bufio.Reader) string {
	t.Helper()

	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := r.ReadString('\n')
		ch <- result{line: line, err: err}
	}()

	select {
	case got := <-ch:
		require.NoError(t, got.err)

		return got.line
	case <-time.After(waitTimeout):
		t.Fatal("SSE の行が届かなかった")

		return ""
	}
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ値の設定は既定値に寄る", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})

			assert.Equal(t, DefaultMaxConnections, reg.set.MaxConnections)
			assert.Equal(t, DefaultReplayConcurrency, reg.set.ReplayConcurrency)
		})

		t.Run("replay の枠は設定した本数だけ確保される", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{ReplayConcurrency: 1})

			require.True(t, reg.sem.TryAcquire(1))
			assert.False(t, reg.sem.TryAcquire(1), "枠が 1 本しか無いこと")
		})
	})
}

func TestRegistry_Stream(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("確定後に cursor より後ろの event を書く", func(t *testing.T) {
			t.Parallel()

			reg, replayer, _ := testRegistry(t, Settings{})
			replayer.EXPECT().ReadPage(gomock.Any(), rt.StreamID("stream-1"), rt.Sequence(0)).
				Return([]rt.DeliveryEvent{testEvent(1)}, false, nil)
			replayer.EXPECT().ReadPage(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, false, nil).AnyTimes()

			r := serveStream(t, reg)

			assert.Equal(t, "id: 1\n", readLine(t, r))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続数の上限に達していれば確定前に断り Retry-After を添える", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{MaxConnections: 1})
			_, err := reg.register(testRequest)
			require.NoError(t, err)

			rec, err := callStream(t, reg)

			require.ErrorIs(t, err, ErrConnectionCapacity)
			assert.Equal(t, "5", rec.Header().Get("Retry-After"))
			assert.Empty(t, rec.Body.String(), "レスポンスを確定していないこと")
		})

		t.Run("初回 replay の枠が空かなければ確定前に断る", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{ReplayConcurrency: 1})
			require.True(t, reg.sem.TryAcquire(1))

			rec, err := callStream(t, reg)

			require.ErrorIs(t, err, ErrReplayAdmission)
			assert.Equal(t, "5", rec.Header().Get("Retry-After"))
		})

		t.Run("登録の直後に ticket が無効なら確定前に断る", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/streams/stream-1", nil)
			rec := httptest.NewRecorder()
			in := testRequest
			in.Revalidate = func(context.Context) error { return ucrealtime.ErrTicketInvalid }

			err := reg.Stream(e.NewContext(req, rec), in)

			require.ErrorIs(t, err, ucrealtime.ErrTicketInvalid)
			assert.Empty(t, rec.Body.String(), "レスポンスを確定していないこと")
			assert.Empty(t, reg.conns, "断った接続は索引に残さないこと")
		})

		t.Run("停止に入っていれば新規接続を受け付けない", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			reg.startDraining()

			_, err := callStream(t, reg)

			require.ErrorIs(t, err, ErrDraining)
		})
	})
}

func TestRegistry_Wake(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("その stream の接続にだけ位置を伝える", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			target, err := reg.register(testRequest)
			require.NoError(t, err)
			other, err := reg.register(StreamRequest{Subject: "subject-2", Destination: "stream-2"})
			require.NoError(t, err)

			reg.Wake(context.Background(), "stream-1", 12)

			assert.Equal(t, int64(12), target.pending.Load())
			assert.Equal(t, int64(0), other.pending.Load(), "別 stream の接続は起こさないこと")
		})

		t.Run("繋がっていない stream への通知は何も起こさない", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})

			assert.NotPanics(t, func() { reg.Wake(context.Background(), "stream-unknown", 1) })
		})
	})
}

func TestRegistry_Revoke(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("その subject の該当 destination だけを STOP で閉じる", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			revoked, err := reg.register(testRequest)
			require.NoError(t, err)
			kept, err := reg.register(StreamRequest{Subject: "subject-1", Destination: "stream-2"})
			require.NoError(t, err)

			reg.Revoke(context.Background(), "subject-1", "stream-1")

			<-revoked.quit
			assert.Equal(t, closeReasonRevoked, revoked.reason)
			got, ok := revoked.takeControl()
			require.True(t, ok)
			assert.Equal(t, "STOP", string(got.Action))

			select {
			case <-kept.quit:
				t.Fatal("同じ subject でも destination が違う接続は閉じないこと")
			default:
			}
		})
	})
}

func TestRegistry_Drain(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続が無ければ即座に終わる", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})

			assert.NoError(t, reg.Drain(context.Background()))
		})

		t.Run("RECONNECT を積んで閉じ、終わるまで待つ", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			conn, err := reg.register(testRequest)
			require.NoError(t, err)
			go func() {
				<-conn.quit
				reg.unregister(context.Background(), conn)
			}()

			require.NoError(t, reg.Drain(context.Background()))

			assert.Equal(t, closeReasonDraining, conn.reason)
			got, ok := conn.takeControl()
			require.True(t, ok)
			assert.Equal(t, "RECONNECT", string(got.Action))
		})

		t.Run("猶予を超えたら残りを諦めて返る", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			_, err := reg.register(testRequest)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			t.Cleanup(cancel)

			assert.NoError(t, reg.Drain(ctx), "閉じ切れなくても HTTP shutdown を待たせないこと")
		})

		t.Run("停止後は新規接続を受け付けない", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			require.NoError(t, reg.Drain(context.Background()))

			_, err := reg.register(testRequest)

			require.ErrorIs(t, err, ErrDraining)
		})
	})
}

func TestRegistry_pump(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("buffer の event を書き、積まれた指示を送って終わる", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			conn := testConn()
			conn.events <- testEvent(1)

			finished := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
				defer close(finished)
				w := newSSEWriter(res, writeDeadline)
				// handler goroutine では require を使えない（testifylint go-require）。
				if !assert.NoError(t, w.commit()) {
					return
				}
				reg.pump(r.Context(), conn, w)
			}))
			res := doStreamRequest(t, srv)
			t.Cleanup(func() { _ = res.Body.Close(); srv.Close() })
			r := bufio.NewReader(res.Body)

			assert.Equal(t, "id: 1\n", readLine(t, r))
			assert.Contains(t, readLine(t, r), `"sequence":"1"`)
			assert.Equal(t, "\n", readLine(t, r), "フレームは空行で終わること")

			conn.closeWith(stopControl(), closeReasonRevoked)

			assert.Equal(t, "event: control\n", readLine(t, r))
			<-finished
		})

		t.Run("heartbeat の周期で id を持たない comment を書く", func(t *testing.T) {
			t.Parallel()

			reg, _, sleeper := testRegistry(t, Settings{})
			conn := testConn()

			finished := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
				defer close(finished)
				w := newSSEWriter(res, writeDeadline)
				// handler goroutine では require を使えない（testifylint go-require）。
				if !assert.NoError(t, w.commit()) {
					return
				}
				reg.pump(r.Context(), conn, w)
			}))
			res := doStreamRequest(t, srv)
			t.Cleanup(func() { conn.close(closeReasonCanceled); _ = res.Body.Close(); srv.Close(); <-finished })
			r := bufio.NewReader(res.Body)

			sleeper.tick(t, tickHeartbeat)

			assert.Equal(t, ": heartbeat\n", readLine(t, r))
		})

		t.Run("寿命に達したら REAUTHENTICATE を送って終わる", func(t *testing.T) {
			t.Parallel()

			reg, _, sleeper := testRegistry(t, Settings{})
			conn := testConn()

			finished := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
				defer close(finished)
				w := newSSEWriter(res, writeDeadline)
				// handler goroutine では require を使えない（testifylint go-require）。
				if !assert.NoError(t, w.commit()) {
					return
				}
				reg.pump(r.Context(), conn, w)
			}))
			res := doStreamRequest(t, srv)
			t.Cleanup(func() { _ = res.Body.Close(); srv.Close() })
			r := bufio.NewReader(res.Body)

			sleeper.tick(t, tickLifetime)

			assert.Equal(t, "event: control\n", readLine(t, r))
			assert.Contains(t, readLine(t, r), "AUTH_REFRESH_REQUIRED")
			<-finished
			assert.Equal(t, closeReasonLifetime, conn.reason)
		})
	})
}

func TestRegistry_writeOrClose(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("書けていれば続ける", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			conn := testConn()

			assert.True(t, reg.writeOrClose(conn, nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限切れは追いつけない client として閉じる", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			conn := testConn()

			assert.False(t, reg.writeOrClose(conn, timeoutError{}))
			assert.Equal(t, closeReasonSlowClient, conn.reason)
		})
	})
}

func TestRegistry_register(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("stream と subject の両方の索引に載る", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})

			conn, err := reg.register(testRequest)

			require.NoError(t, err)
			assert.Len(t, reg.byStream["stream-1"], 1)
			assert.Len(t, reg.bySubject["subject-1"], 1)
			assert.Equal(t, conn, reg.conns[conn.id])
		})

		t.Run("接続ごとに違う番号が振られる", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			first, err := reg.register(testRequest)
			require.NoError(t, err)
			second, err := reg.register(testRequest)
			require.NoError(t, err)

			assert.NotEqual(t, first.id, second.id)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限に達していれば ErrConnectionCapacity を返す", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{MaxConnections: 1})
			_, err := reg.register(testRequest)
			require.NoError(t, err)

			_, err = reg.register(testRequest)

			require.ErrorIs(t, err, ErrConnectionCapacity)
		})
	})
}

func TestRegistry_unregister(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("索引から外して容量を返し終わりを知らせる", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			conn, err := reg.register(testRequest)
			require.NoError(t, err)

			reg.unregister(context.Background(), conn)

			<-conn.done
			assert.Empty(t, reg.conns)
			assert.Empty(t, reg.byStream, "空になった区画ごと消えること")
			assert.Empty(t, reg.bySubject)
		})

		t.Run("既に閉じている接続の理由は上書きしない", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			conn, err := reg.register(testRequest)
			require.NoError(t, err)
			conn.close(closeReasonRevoked)

			reg.unregister(context.Background(), conn)

			assert.Equal(t, closeReasonRevoked, conn.reason)
		})
	})
}

func TestRegistry_admit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("枠が空いていれば確保する", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{ReplayConcurrency: 1})

			require.NoError(t, reg.admit(context.Background()))
			assert.False(t, reg.sem.TryAcquire(1), "枠を確保したまま返ること")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("待っても空かなければ ErrReplayAdmission を返す", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{ReplayConcurrency: 1})
			require.True(t, reg.sem.TryAcquire(1))

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			t.Cleanup(cancel)

			require.ErrorIs(t, reg.admit(ctx), ErrReplayAdmission)
		})
	})
}

func TestRegistry_startDraining(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受付を止めてそのとき保持していた接続を返す", func(t *testing.T) {
			t.Parallel()

			reg, _, _ := testRegistry(t, Settings{})
			conn, err := reg.register(testRequest)
			require.NoError(t, err)

			got := reg.startDraining()

			require.Len(t, got, 1)
			assert.Equal(t, conn, got[0])
			assert.True(t, reg.draining)
		})
	})
}

func Test_values(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("区画の接続を slice にする", func(t *testing.T) {
			t.Parallel()

			conn := testConn()

			got := values(map[uint64]*connection{conn.id: conn})

			require.Len(t, got, 1)
			assert.Equal(t, conn, got[0])
		})

		t.Run("空の区画は空の slice になる", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, values(nil))
		})
	})
}

func Test_addIndex(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("区画が無ければ作って加える", func(t *testing.T) {
			t.Parallel()

			m := map[rt.StreamID]map[uint64]*connection{}
			conn := testConn()

			addIndex(m, "stream-1", conn)

			assert.Equal(t, conn, m["stream-1"][conn.id])
		})
	})
}

func Test_removeIndex(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最後の 1 本を外すと区画ごと消える", func(t *testing.T) {
			t.Parallel()

			m := map[rt.StreamID]map[uint64]*connection{}
			conn := testConn()
			addIndex(m, "stream-1", conn)

			removeIndex(m, "stream-1", conn.id)

			assert.Empty(t, m, "stream の数に限りが無いので区画を残さないこと")
		})

		t.Run("知らない key を外しても壊れない", func(t *testing.T) {
			t.Parallel()

			m := map[rt.StreamID]map[uint64]*connection{}

			assert.NotPanics(t, func() { removeIndex(m, "stream-unknown", 1) })
		})
	})
}
