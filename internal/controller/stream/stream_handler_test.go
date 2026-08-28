package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/stream/gen"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	mock_realtime "go-boilerplate/internal/usecase/realtime/mock"
	"go-boilerplate/pkg/xerrors"
)

const targetPath = "/v1/streams/:destination"

var errStreamClosed = xerrors.New("stream closed")

// stubStreamer は、handler が渡した StreamRequest を捕まえる Streamer です。
type stubStreamer struct {
	got    *StreamRequest
	gotCtx context.Context
	err    error
	called bool
}

func (s *stubStreamer) Stream(c *echo.Context, req StreamRequest) error {
	s.called = true
	s.got = &req
	s.gotCtx = c.Request().Context()
	return s.err
}

func newServer(t *testing.T) (*server, *mock_realtime.MockCursorValidator, *stubStreamer) {
	t.Helper()
	cursors := mock_realtime.NewMockCursorValidator(gomock.NewController(t))
	streamer := &stubStreamer{}
	tp, _ := observability.NewRecordingTracerProvider(t)
	s := &server{tracer: observability.NewTracerFactory(tp).Controller(), cursors: cursors, streamer: streamer}
	return s, cursors, streamer
}

// newContext は、StreamGrant スロットを仕込んだ request の echo.Context を返します。grant が nil ならスロットは空のままです。
func newContext(t *testing.T, target string, grant *rt.StreamGrant) *echo.Context {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	req = req.WithContext(ctxhelper.WithStreamGrant(req.Context()))
	if grant != nil {
		require.True(t, ctxhelper.SetStreamGrant(req.Context(), *grant))
	}
	return echo.New().NewContext(req, httptest.NewRecorder())
}

// grantFor は、stream-1 に束縛された検証済み ticket を返します。
func grantFor() *rt.StreamGrant {
	return &rt.StreamGrant{Subject: "subject-1", Destination: "stream-1", Scope: "read", InitialCursor: 3}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	BindHandler(e, observability.NewNoopTracerFactory(t), mock_realtime.NewMockCursorValidator(gomock.NewController(t)), &stubStreamer{})

	routes := e.Router().Routes()

	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, targetPath, routes[0].Path)
}

func Test_server_GetStream(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Last-Event-IDを最優先の位置として検証しStreamerへ渡す", func(t *testing.T) {
			t.Parallel()
			s, cursors, streamer := newServer(t)
			cursors.EXPECT().Validate(gomock.Any(), rt.StreamID("stream-1"), rt.Sequence(12)).Return(nil)
			lastEventID, after := "12", "7"

			c := newContext(t, "/v1/streams/stream-1?after=7", grantFor())
			original := c.Request().Context()
			err := s.GetStream(c, "stream-1", gen.GetStreamParams{After: &after, LastEventID: &lastEventID})

			require.NoError(t, err)
			require.NotNil(t, streamer.got)
			assert.Equal(t, StreamRequest{Subject: "subject-1", Destination: "stream-1", Scope: "read", Cursor: 12}, *streamer.got)
			// handler が張った span 付きの context が request に据え直され、その request が Streamer に渡ること。
			assert.NotEqual(t, original, streamer.gotCtx)
			assert.Equal(t, c.Request().Context(), streamer.gotCtx)
		})

		t.Run("afterだけならその位置を使う", func(t *testing.T) {
			t.Parallel()
			s, cursors, streamer := newServer(t)
			cursors.EXPECT().Validate(gomock.Any(), rt.StreamID("stream-1"), rt.Sequence(7)).Return(nil)
			after := "7"

			require.NoError(t, s.GetStream(newContext(t, "/v1/streams/stream-1?after=7", grantFor()), "stream-1", gen.GetStreamParams{After: &after}))

			assert.Equal(t, rt.Sequence(7), streamer.got.Cursor)
		})

		t.Run("どちらも無ければticketの初期位置から始める", func(t *testing.T) {
			t.Parallel()
			s, cursors, streamer := newServer(t)
			cursors.EXPECT().Validate(gomock.Any(), rt.StreamID("stream-1"), rt.Sequence(3)).Return(nil)

			require.NoError(t, s.GetStream(newContext(t, "/v1/streams/stream-1", grantFor()), "stream-1", gen.GetStreamParams{}))

			assert.Equal(t, rt.Sequence(3), streamer.got.Cursor)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("検証済みticketが無ければErrUnauthenticatedでStreamerを呼ばない", func(t *testing.T) {
			t.Parallel()
			s, _, streamer := newServer(t)

			err := s.GetStream(newContext(t, "/v1/streams/stream-1", nil), "stream-1", gen.GetStreamParams{})

			require.ErrorIs(t, err, ctxhelper.ErrStreamGrantMissing)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
			assert.False(t, streamer.called)
		})

		t.Run("ticketの束縛と違うdestinationはErrTicketInvalid", func(t *testing.T) {
			t.Parallel()
			s, _, streamer := newServer(t)

			err := s.GetStream(newContext(t, "/v1/streams/stream-2", grantFor()), "stream-2", gen.GetStreamParams{})

			require.ErrorIs(t, err, ucrealtime.ErrTicketInvalid)
			assert.False(t, streamer.called)
		})

		t.Run("cursorの形式不正はErrCursorMalformedにINVALID_STREAM_CURSORを添えて返す", func(t *testing.T) {
			t.Parallel()
			s, _, streamer := newServer(t)
			after := "-1"

			err := s.GetStream(newContext(t, "/v1/streams/stream-1?after=-1", grantFor()), "stream-1", gen.GetStreamParams{After: &after})

			require.ErrorIs(t, err, ErrCursorMalformed)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, codeInvalidStreamCursor, meta.Code())
			assert.False(t, streamer.called)
		})

		t.Run("replay floorより前のcursorはErrGoneにSTREAM_CURSOR_EXPIREDを添えて返す", func(t *testing.T) {
			t.Parallel()
			s, cursors, streamer := newServer(t)
			cursors.EXPECT().Validate(gomock.Any(), rt.StreamID("stream-1"), rt.Sequence(3)).
				Return(xerrors.Wrap(ucrealtime.ErrCursorExpired, "gone"))

			err := s.GetStream(newContext(t, "/v1/streams/stream-1", grantFor()), "stream-1", gen.GetStreamParams{})

			require.ErrorIs(t, err, apperror.ErrGone)
			require.ErrorIs(t, err, ucrealtime.ErrCursorExpired)
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, codeStreamCursorExpired, meta.Code())
			assert.False(t, streamer.called)
		})

		t.Run("EventLogの不達はErrUnavailableのまま返す", func(t *testing.T) {
			t.Parallel()
			s, cursors, streamer := newServer(t)
			cursors.EXPECT().Validate(gomock.Any(), rt.StreamID("stream-1"), rt.Sequence(3)).Return(apperror.ErrUnavailable)

			err := s.GetStream(newContext(t, "/v1/streams/stream-1", grantFor()), "stream-1", gen.GetStreamParams{})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotErrorIs(t, err, apperror.ErrGone)
			assert.False(t, streamer.called)
		})

		t.Run("Streamerのエラーはそのまま返す", func(t *testing.T) {
			t.Parallel()
			s, cursors, streamer := newServer(t)
			cursors.EXPECT().Validate(gomock.Any(), rt.StreamID("stream-1"), rt.Sequence(3)).Return(nil)
			streamer.err = errStreamClosed

			err := s.GetStream(newContext(t, "/v1/streams/stream-1", grantFor()), "stream-1", gen.GetStreamParams{})

			require.ErrorIs(t, err, errStreamClosed)
		})
	})
}
