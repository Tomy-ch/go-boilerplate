package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
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
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	// streamTestTicket は、stub の verifier が受け入れる唯一の ticket 生値（テスト用のダミー）。
	streamTestTicket = "stream-ticket-accepted-by-stub"
	// streamTestDestination は、ticket が束縛された stream。
	streamTestDestination = rt.StreamID("stream-1")
	// streamTestFloor は、stub の CursorValidator が replay floor とみなす位置。これより前の cursor は失効。
	streamTestFloor = rt.Sequence(5)
	// streamTestInitialCursor は、ticket に許可された初期位置。
	streamTestInitialCursor = rt.Sequence(7)
)

// stubTicketVerifier は、固定の生値 × destination だけを受け入れる TicketVerifier。それ以外は ErrTicketInvalid。
type stubTicketVerifier struct{}

// stubCursorValidator は、streamTestFloor より前の cursor を失効とみなす CursorValidator。unavailable なら常に ErrUnavailable。
type stubCursorValidator struct {
	unavailable bool
}

// stubStreamer は、検証を通った接続の要求をそのまま JSON で返す Streamer（本物は Phase 6）。
type stubStreamer struct{}

func (stubTicketVerifier) Verify(_ context.Context, value string, destination rt.StreamID) (ucrealtime.VerifiedTicketView, error) {
	if value != streamTestTicket || destination != streamTestDestination {
		return ucrealtime.VerifiedTicketView{}, ucrealtime.ErrTicketInvalid
	}
	return ucrealtime.VerifiedTicketView{
		Subject: "subject-1", Destination: streamTestDestination, Scope: "read", InitialCursor: streamTestInitialCursor,
	}, nil
}

func (v stubCursorValidator) Validate(_ context.Context, _ rt.StreamID, cursor rt.Sequence) error {
	if v.unavailable {
		return xerrors.Wrap(apperror.ErrUnavailable, "event log unreachable")
	}
	if cursor < streamTestFloor {
		return xerrors.Wrap(ucrealtime.ErrCursorExpired, "below the replay floor")
	}
	return nil
}

func (stubStreamer) Stream(c *echo.Context, req stream.StreamRequest) error {
	return c.JSON(http.StatusOK, map[string]string{
		"subject": req.Subject, "destination": string(req.Destination), "scope": req.Scope, "cursor": req.Cursor.String(),
	})
}

// newStreamServer は、本物の OpenAPI validator（実 spec）・StreamTicket scheme・stream handler・アクセスログ・
// エラーハンドラを本番と同じ順で配線した Echo を起動する。ticket の検証と replay floor だけを stub に置き換える。
func newStreamServer(t *testing.T, logger logging.Logger, cursors ucrealtime.CursorValidator) *Server {
	t.Helper()

	spec, err := validator.GetValidator()
	require.NoError(t, err)

	bearer := mock_auth.NewMockAuthenticator(gomock.NewController(t))
	authFunc := auth.NewAuthenticator(bearer, stubIdentityResolver{}, []auth.SchemeAuthenticator{streamauth.New(stubTicketVerifier{})})

	e := echo.New()
	UseAppErrorHandlerWithLogger(t, e, logger,
		instrumentation.LoggingMiddleware(logger, logging.NewTestLogFieldBuilder(t), redaction.FromSpec(spec)).Middleware,
		inbound.OpenAPIMiddleware(spec, skipper.New(), authFunc).Middleware,
	)
	stream.BindHandler(e, observability.NewNoopTracerFactory(t), cursors, stubStreamer{})

	return StartServer(t, e)
}

func decodeStreamBody(t *testing.T, res *http.Response) map[string]string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	return body
}

func TestStreamAuth_Integration(t *testing.T) {
	t.Parallel()

	srv := newStreamServer(t, logging.NewTestLogger(t), stubCursorValidator{})

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なticketとafterで接続が受け入れられStreamerへ到達する", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket="+streamTestTicket+"&after=10", nil, nil)
			require.Equal(t, http.StatusOK, res.StatusCode)

			body := decodeStreamBody(t, res)
			assert.Equal(t, "subject-1", body["subject"])
			assert.Equal(t, "stream-1", body["destination"])
			assert.Equal(t, "10", body["cursor"])
		})

		t.Run("Last-Event-IDはafterより優先される", func(t *testing.T) {
			t.Parallel()
			headers := http.Header{}
			headers.Set("Last-Event-ID", "12")
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket="+streamTestTicket+"&after=10", nil, headers)
			require.Equal(t, http.StatusOK, res.StatusCode)

			assert.Equal(t, "12", decodeStreamBody(t, res)["cursor"])
		})

		t.Run("cursorが無ければticketの初期位置から始まる", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket="+streamTestTicket, nil, nil)
			require.Equal(t, http.StatusOK, res.StatusCode)

			assert.Equal(t, streamTestInitialCursor.String(), decodeStreamBody(t, res)["cursor"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ticketが無ければ401", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?after=10", nil, nil)
			errResp := AssertErrorResponseBody(t, res, http.StatusUnauthorized)
			assert.Equal(t, "UNAUTHORIZED", errResp.Code)
		})

		t.Run("不明なticketは401", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket=not-the-accepted-one", nil, nil)
			errResp := AssertErrorResponseBody(t, res, http.StatusUnauthorized)
			assert.Equal(t, "UNAUTHORIZED", errResp.Code)
		})

		t.Run("ticketが束縛されたdestination以外への接続は401", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-2?ticket="+streamTestTicket, nil, nil)
			errResp := AssertErrorResponseBody(t, res, http.StatusUnauthorized)
			assert.Equal(t, "UNAUTHORIZED", errResp.Code)
		})

		t.Run("specのpatternに合わないafterはvalidatorが400で拒否する", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket="+streamTestTicket+"&after=abc", nil, nil)
			errResp := AssertErrorResponseBody(t, res, http.StatusBadRequest)
			assert.Equal(t, "BAD_REQUEST", errResp.Code)
		})

		t.Run("int64を超えるafterはINVALID_STREAM_CURSORの400", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket="+streamTestTicket+"&after=9999999999999999999", nil, nil)
			errResp := AssertErrorResponseBody(t, res, http.StatusBadRequest)
			assert.Equal(t, "INVALID_STREAM_CURSOR", errResp.Code)
		})

		t.Run("replay floorより前のafterはSTREAM_CURSOR_EXPIREDの410", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket="+streamTestTicket+"&after=1", nil, nil)
			errResp := AssertErrorResponseBody(t, res, http.StatusGone)
			assert.Equal(t, "STREAM_CURSOR_EXPIRED", errResp.Code)
		})

		t.Run("EventLogが読めなければ503", func(t *testing.T) {
			t.Parallel()
			degraded := newStreamServer(t, logging.NewTestLogger(t), stubCursorValidator{unavailable: true})
			res := degraded.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket="+streamTestTicket+"&after=10", nil, nil)
			errResp := AssertErrorResponseBody(t, res, http.StatusServiceUnavailable)
			assert.Equal(t, "SERVICE_UNAVAILABLE", errResp.Code)
		})
	})
}
