package integration

import (
	"context"
	"net/http"
	"testing"

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
	"go-boilerplate/internal/infrastructure/realtimesecret"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_auth "go-boilerplate/internal/usecase/boundary/auth/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	rttestkit "go-boilerplate/internal/usecase/boundary/realtime/testkit"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

// revocationFixture は、失効を発行側から駆動するのに要る一式です。stream_sse_test.go の fixture が
// ticket を stub で通すのに対し、こちらは実物の store・issuer・verifier・AccessRevoker を繋ぎます。
// 失効の検証は「無効化」と「通知」が両方効いて初めて成立し、片方だけでは基準を満たせないためです。
type revocationFixture struct {
	srv      *Server
	tickets  *rttestkit.StreamTicketStore
	issuer   ucrealtime.TicketIssuer
	revoker  ucrealtime.AccessRevoker
	registry *stream.Registry
}

// fanoutToRegistry は、SNS / SQS の往復の代わりに失効通知をこの instance の registry へ直接渡します。
// 往復そのものは infrastructure/realtime/aws の contract test が固定しており、ここで見たいのは
// AccessRevoker が無効化と通知の両方を起こすことです。
type fanoutToRegistry struct {
	registry *stream.Registry
}

func (f fanoutToRegistry) NotifyRevoked(ctx context.Context, subject string, destination rt.StreamID) error {
	f.registry.Revoke(ctx, subject, destination)

	return nil
}

func newRevocationServer(t *testing.T) *revocationFixture {
	t.Helper()

	spec, err := validator.GetValidator()
	require.NoError(t, err)

	tickets := rttestkit.NewStreamTicketStore()
	clk := clocktestkit.NewMockClock(t, sseNow)
	tf := observability.NewNoopTracerFactory(t)
	logger := logging.NewTestLogger(t)

	verifier := ucrealtime.NewTicketVerifier(tickets, clk, tf)
	bearer := mock_auth.NewMockAuthenticator(gomock.NewController(t))
	authFunc, err := auth.NewAuthenticator(
		bearer, stubIdentityResolver{}, []auth.SchemeAuthenticator{streamauth.New(verifier)},
	)
	require.NoError(t, err)

	replayLog := rttestkit.NewEventLog()
	seedEvents(replayLog, 1)
	cursorLog := rttestkit.NewEventLog()
	seedEvents(cursorLog, 1)

	registry := stream.NewRegistry(ucrealtime.NewReplayer(replayLog, tf), newSSESleeper(), logger, tf,
		observability.NewNoopRealtimeMetrics(t), ucrealtime.NewHealth(replayLog), stream.Settings{})

	e := echo.New()
	UseAppErrorHandlerWithLogger(t, e, logger,
		instrumentation.LoggingMiddleware(logger, logging.NewTestLogFieldBuilder(t), redaction.FromSpec(spec)).Middleware,
		inbound.OpenAPIMiddleware(spec, skipper.New(), authFunc).Middleware,
	)
	stream.BindHandler(e, tf, ucrealtime.NewCursorValidator(cursorLog, clk, tf), registry)

	return &revocationFixture{
		srv:      StartServer(t, e),
		tickets:  tickets,
		issuer:   ucrealtime.NewTicketIssuer(tickets, realtimesecret.New(), clk, tf),
		revoker:  ucrealtime.NewAccessRevoker(tickets, fanoutToRegistry{registry: registry}, tf),
		registry: registry,
	}
}

// issue は、subject へ destination の ticket を発行し、生値を返します。
func (f *revocationFixture) issue(t *testing.T, subject string) string {
	t.Helper()

	view, err := f.issuer.Issue(context.Background(), ucrealtime.IssueTicketInput{
		Subject: subject, Destination: streamTestDestination, InitialCursor: 0,
	})
	require.NoError(t, err)

	return view.Value
}

func TestStreamRevocation_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失効させると接続が STOP で閉じ、同じ ticket では繋ぎ直せない", func(t *testing.T) {
			t.Parallel()

			f := newRevocationServer(t)
			value := f.issue(t, "subject-1")
			path := "/v1/streams/" + string(streamTestDestination) + "?ticket=" + value

			c, res := connectSSE(t, f.srv, path)
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "1", c.next().id)

			require.NoError(t, f.revoker.Revoke(context.Background(), "subject-1", streamTestDestination))

			frame := c.next()
			assert.Equal(t, "control", frame.event)
			require.NotNil(t, c.lastControl)
			assert.Equal(t, "STOP", string(c.lastControl.Action))
			assert.True(t, c.closedBySelf, "STOP を受けた client は自ら閉じること")

			// STOP を無視する client が同じ ticket で戻ってこられないこと。無効化が効いていなければここが 200 になる。
			_, again := connectSSE(t, f.srv, path)
			assert.Equal(t, http.StatusUnauthorized, again.StatusCode)
			assert.Zero(t, f.tickets.Len(), "失効した ticket は store に残らないこと")
		})

		t.Run("失効は subject と destination の組にだけ効く", func(t *testing.T) {
			t.Parallel()

			f := newRevocationServer(t)
			mine := f.issue(t, "subject-1")
			other := f.issue(t, "subject-2")

			require.NoError(t, f.revoker.Revoke(context.Background(), "subject-2", streamTestDestination))

			_, res := connectSSE(t, f.srv, "/v1/streams/"+string(streamTestDestination)+"?ticket="+mine)
			assert.Equal(t, http.StatusOK, res.StatusCode, "他人の失効で自分の ticket が巻き添えにならないこと")
			_, revoked := connectSSE(t, f.srv, "/v1/streams/"+string(streamTestDestination)+"?ticket="+other)
			assert.Equal(t, http.StatusUnauthorized, revoked.StatusCode)
		})
	})
}
