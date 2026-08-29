package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/controller/httpstack/oapi/auth"
	"go-boilerplate/internal/controller/httpstack/oapi/skipper"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/controller/httpstack/redaction"
	"go-boilerplate/internal/controller/stream"
	streamauth "go-boilerplate/internal/controller/stream/auth"
	"go-boilerplate/internal/di/server/extension/inbound"
	"go-boilerplate/internal/di/server/extension/instrumentation"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_auth "go-boilerplate/internal/usecase/boundary/auth/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	rttestkit "go-boilerplate/internal/usecase/boundary/realtime/testkit"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

// sseNow は、EventLog に置く event の発生時刻であり、cursor の失効判定が見る現在時刻でもあります。
var sseNow = time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

// 接続の周期処理の待ち時間を見分ける帯の境界。controller/stream の定数は非公開なので、
// heartbeat(15s) / catch-up(30s + jitter) / lifetime(1h) が分かれる位置に取ります。
const (
	sseHeartbeatCeiling = 20 * time.Second
	sseCatchUpCeiling   = 5 * time.Minute
)

// sseSleeper は、接続の周期処理を止めておく Sleeper です。catch-up の周期だけテストが名指しで進め、
// heartbeat と connection lifetime は ctx が終わるまで起きません。実時間には一切依存しません。
type sseSleeper struct {
	catchUp chan struct{}
}

func newSSESleeper() *sseSleeper {
	return &sseSleeper{catchUp: make(chan struct{})}
}

// Sleep は、待ち時間の帯で tick の種類を見分け、catch-up だけテストの合図を待ちます。
func (s *sseSleeper) Sleep(ctx context.Context, d time.Duration) error {
	if d <= sseHeartbeatCeiling || d > sseCatchUpCeiling {
		<-ctx.Done()

		return ctx.Err()
	}

	select {
	case <-s.catchUp:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// tickCatchUp は、周期 catch-up を 1 回だけ進めます。
func (s *sseSleeper) tickCatchUp(t *testing.T) {
	t.Helper()

	select {
	case s.catchUp <- struct{}{}:
	case <-time.After(sseReadTimeout):
		t.Fatal("catch-up の待ちが起きなかった")
	}
}

// sseEvent は、seq の位置の封筒を組み立てます。
func sseEvent(seq rt.Sequence) rt.DeliveryEvent {
	return rt.DeliveryEvent{
		EventID:       "evt-" + seq.String(),
		StreamID:      streamTestDestination,
		Sequence:      seq,
		Type:          "sample.thing.created.v1",
		OccurredAt:    sseNow,
		SchemaVersion: 1,
		Payload:       json.RawMessage(`{"n":` + seq.String() + `}`),
	}
}

// seedEvents は、1 から n までの連続した event を EventLog に置きます。
func seedEvents(log *rttestkit.EventLog, n int) {
	events := make([]rt.DeliveryEvent, 0, n)
	for i := 1; i <= n; i++ {
		events = append(events, sseEvent(rt.Sequence(i)))
	}

	log.Seed(events...)
}

// sseFixture は、SSE の scenario を駆動するのに要る一式です。
type sseFixture struct {
	srv      *Server
	log      *rttestkit.EventLog
	sleeper  *sseSleeper
	registry *stream.Registry
}

// newSSEServer は、stream_auth_test.go と同じ配線に**本物の connection registry** を Streamer として差した
// サーバーを起動します。差し替えるのは ticket の検証（stub）と EventLog（in-memory fake）だけで、
// OpenAPI validator・security scheme・cursor の失効判定・replay は本物が走ります。
func newSSEServer(t *testing.T, set stream.Settings) *sseFixture {
	t.Helper()

	spec, err := validator.GetValidator()
	require.NoError(t, err)

	bearer := mock_auth.NewMockAuthenticator(gomock.NewController(t))
	authFunc, err := auth.NewAuthenticator(
		bearer, stubIdentityResolver{}, []auth.SchemeAuthenticator{streamauth.New(stubTicketVerifier{})},
	)
	require.NoError(t, err)

	log := rttestkit.NewEventLog()
	sleeper := newSSESleeper()
	tf := observability.NewNoopTracerFactory(t)
	logger := logging.NewTestLogger(t)

	cursors := ucrealtime.NewCursorValidator(log, clocktestkit.NewMockClock(t, sseNow), tf)
	registry := stream.NewRegistry(ucrealtime.NewReplayer(log, tf), sleeper, logger, tf, set)

	e := echo.New()
	UseAppErrorHandlerWithLogger(t, e, logger,
		instrumentation.LoggingMiddleware(logger, logging.NewTestLogFieldBuilder(t), redaction.FromSpec(spec)).Middleware,
		inbound.OpenAPIMiddleware(spec, skipper.New(), authFunc).Middleware,
	)
	stream.BindHandler(e, tf, cursors, registry)

	return &sseFixture{srv: StartServer(t, e), log: log, sleeper: sleeper, registry: registry}
}

// streamPath は、ticket 付きの接続先を組み立てます。
func streamPath(after rt.Sequence) string {
	return "/v1/streams/" + string(streamTestDestination) + "?ticket=" + streamTestTicket + "&after=" + after.String()
}

func TestStreamSSE_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続すると cursor より後ろの event が sequence 順に届く", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{})
			seedEvents(f.log, 3)

			c, res := connectSSE(t, f.srv, streamPath(1))

			require.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))
			assert.Equal(t, "no-store", res.Header.Get("Cache-Control"))
			assert.Equal(t, "2", c.next().id)
			assert.Equal(t, "3", c.next().id)
		})

		t.Run("afterとLast-Event-IDのどちらでも途中から再開できる", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{})
			seedEvents(f.log, 4)

			byAfter, res := connectSSE(t, f.srv, streamPath(3))
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "4", byAfter.next().id)
			byAfter.close()

			// 同じ位置を Last-Event-ID で提示すると同じ続きが届く（優先順は Phase 5 で固定済み）。
			headers := http.Header{}
			headers.Set("Last-Event-ID", "3")
			byHeader, res := connectSSEWithHeaders(t, f.srv, streamPath(0), headers)

			require.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "4", byHeader.next().id)
		})

		t.Run("wakeupが届かなくても周期catch-upで配信される", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{})
			seedEvents(f.log, 1)

			c, res := connectSSE(t, f.srv, streamPath(0))
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "1", c.next().id)

			// wakeup は一切送らず、後から届いた event を catch-up だけで拾わせる。
			f.log.Seed(sseEvent(2))
			f.sleeper.tickCatchUp(t)

			assert.Equal(t, "2", c.next().id)
		})

		t.Run("接続が閉じた後も再接続すれば取りこぼした event が届く", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{})
			seedEvents(f.log, 5)

			first, res := connectSSE(t, f.srv, streamPath(0))
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "1", first.next().id)
			first.close() // 2 件目以降を読まずに切る

			second, res := connectSSE(t, f.srv, streamPath(parseSequence(t, first.lastEventID)))

			require.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "2", second.next().id, "受け取っていない event は EventLog に残っていること")
		})

		t.Run("停止に入ると RECONNECT を送って閉じ、新規接続は 503 になる", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{})
			seedEvents(f.log, 1)
			c, res := connectSSE(t, f.srv, streamPath(0))
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "1", c.next().id)

			go func() { _ = f.registry.Drain(context.Background()) }()

			frame := c.next()
			assert.Equal(t, "control", frame.event)
			require.NotNil(t, c.lastControl)
			assert.Equal(t, "RECONNECT", string(c.lastControl.Action))
			assert.Equal(t, "SERVER_DRAINING", string(c.lastControl.Reason))

			_, refused := connectSSE(t, f.srv, streamPath(0))
			assert.Equal(t, http.StatusServiceUnavailable, refused.StatusCode)
		})

		t.Run("失効通知を受けた接続は STOP で閉じ client 側が自ら閉じる", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{})
			seedEvents(f.log, 1)
			c, res := connectSSE(t, f.srv, streamPath(0))
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "1", c.next().id)

			f.registry.Revoke(context.Background(), "subject-1", streamTestDestination)

			frame := c.next()
			assert.Equal(t, "control", frame.event)
			assert.Empty(t, frame.id, "control event が Last-Event-ID を汚さないこと")
			assert.Equal(t, "1", c.lastEventID, "cursor は business event のままであること")
			require.NotNil(t, c.lastControl)
			assert.Equal(t, "STOP", string(c.lastControl.Action))
			assert.True(t, c.closedBySelf, "STOP を受けた client は EOF を待たず自ら閉じること")
		})

		t.Run("cursorの続きが失われていれば接続前に410で断られる", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{})
			// 1 と 2 が保持期間から落ち、3 だけが残っている状態。
			f.log.Seed(sseEvent(3))

			_, res := connectSSE(t, f.srv, streamPath(1))

			require.Equal(t, http.StatusGone, res.StatusCode)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続数の上限を超えると503とRetry-Afterで断られ、閉じれば戻る", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{MaxConnections: 3})
			seedEvents(f.log, 1)

			held := make([]*sseClient, 0, 3)
			for range 3 {
				c, res := connectSSE(t, f.srv, streamPath(0))
				require.Equal(t, http.StatusOK, res.StatusCode)
				require.Equal(t, "1", c.next().id)
				held = append(held, c)
			}

			_, refused := connectSSE(t, f.srv, streamPath(0))
			require.Equal(t, http.StatusServiceUnavailable, refused.StatusCode)
			assert.Equal(t, "5", refused.Header.Get("Retry-After"))

			held[0].close()

			require.Eventually(t, func() bool {
				_, res := connectSSE(t, f.srv, streamPath(0))

				return res.StatusCode == http.StatusOK
			}, sseReadTimeout, 20*time.Millisecond, "閉じた接続の容量が回収されること")
		})

		t.Run("初回replayの枠が空かなければ503で断られる", func(t *testing.T) {
			t.Parallel()

			f := newSSEServer(t, stream.Settings{ReplayConcurrency: 1})
			seedEvents(f.log, 1)

			// 1 本目に枠を握らせたまま初回 replay を終わらせない。
			release := f.log.Hold()
			t.Cleanup(release)

			_, res := connectSSE(t, f.srv, streamPath(0))
			require.Equal(t, http.StatusOK, res.StatusCode, "枠は確定前に取れているので 1 本目は繋がる")

			// 枠は 1 本しかないので、2 本目は有界待ちの後に断られる。
			_, refused := connectSSE(t, f.srv, streamPath(0))

			assert.Equal(t, http.StatusServiceUnavailable, refused.StatusCode)
			assert.Equal(t, "5", refused.Header.Get("Retry-After"))
		})
	})
}

// parseSequence は、client が保持している Last-Event-ID を cursor に直します。
func parseSequence(t *testing.T, s string) rt.Sequence {
	t.Helper()

	n, err := strconv.ParseInt(s, 10, 64)
	require.NoError(t, err)

	return rt.Sequence(n)
}
