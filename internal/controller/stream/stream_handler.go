//go:generate oapi-codegen --include-tags=v1/streams --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/streams --package=gen --generate=echo5-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package stream は、SSE stream endpoint（GET /v1/streams/{destination}）の handler です。
// レスポンスを確定する前の判定 — ticket の検証（security scheme が済ませ StreamGrant に置く）、cursor の解決、
// replay floor の確認 — をここで行い、確定後の配信は Streamer が担います。
// strict-server を使わないのは、SSE が 1 つの返り値で表せず、flush と切断の制御を handler が握る必要があるためです。
package stream

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/stream/gen"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

const (
	// codeInvalidStreamCursor は、cursor の形式が不正なときのエラーコードです。
	codeInvalidStreamCursor = "INVALID_STREAM_CURSOR"
	// codeStreamCursorExpired は、cursor が replay floor より前にあるときのエラーコードです。
	codeStreamCursorExpired = "STREAM_CURSOR_EXPIRED"
)

type server struct {
	tracer   observability.LayerTracer
	cursors  ucrealtime.CursorValidator
	streamer Streamer
}

// BindHandler は、stream endpoint の handler を Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, cursors ucrealtime.CursorValidator, streamer Streamer) {
	gen.RegisterHandlers(e, &server{tracer: tf.Controller(), cursors: cursors, streamer: streamer})
}

// GetStream は、接続を検証してから Streamer へ渡します。拒否はすべてレスポンスを確定する前に返します:
// ticket の不備は 401、cursor の形式不正は 400、replay floor より前の cursor は 410、EventLog の不達は 503。
func (s *server) GetStream(c *echo.Context, destination gen.StreamDestinationParam, params gen.GetStreamParams) error {
	ctx, endSpan := s.tracer.Start(c.Request().Context())
	defer endSpan()

	grant, err := ctxhelper.RequireStreamGrant(ctx)
	if err != nil {
		return err
	}

	if grant.Destination != rt.StreamID(destination) {
		return xerrors.Wrap(ucrealtime.ErrTicketInvalid, "ticket is bound to another destination")
	}

	cursor, err := resolveCursor(params.LastEventID, params.After, grant.InitialCursor)
	if err != nil {
		return apperror.WithMeta(err, apperror.NewMeta(codeInvalidStreamCursor))
	}

	if err := s.cursors.Validate(ctx, grant.Destination, cursor); err != nil {
		if xerrors.Is(err, ucrealtime.ErrCursorExpired) {
			return apperror.WithMeta(xerrors.Join(apperror.ErrGone, err), apperror.NewMeta(codeStreamCursorExpired))
		}
		return err
	}

	c.SetRequest(c.Request().WithContext(ctx))

	return s.streamer.Stream(c, StreamRequest{
		Subject: grant.Subject, Destination: grant.Destination, Scope: grant.Scope, Cursor: cursor,
	})
}
