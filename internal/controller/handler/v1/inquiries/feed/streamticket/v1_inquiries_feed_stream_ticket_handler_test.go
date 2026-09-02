package streamticket

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/inquiries/feed/streamticket/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	mock_inquiryuc "go-boilerplate/internal/usecase/inquiry/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

// authnContext は、内部ユーザー ID を解決済みの認証コンテキストを返すテストヘルパーです。
func authnContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	authn, err := auth.New("operator-jane", "issuer", nil, nil)
	require.NoError(t, err)

	resolved, err := authn.WithUserID(userID)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *resolved))
	return ctx
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, observability.NewNoopTracerFactory(t), uc)

	routes := e.Router().Routes()
	testassert.AssertEchoRouterPath(t, "/v1/inquiries/feed/stream-ticket", routes)
	testassert.AssertEchoRouterMethods(t, []string{http.MethodPost}, routes)
}

func Test_server_PostInquiriesFeedStreamTicket(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("発行したticketと接続先を応答へ写す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expiresAt := time.Date(2026, time.September, 1, 10, 5, 0, 0, time.UTC)

			var capturedSubject string
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().IssueFeedTicket(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *auth.Authn) (inquiryuc.TicketView, error) {
					capturedSubject = authn.Subject()
					return inquiryuc.TicketView{
						Ticket: "raw-feed-ticket", StreamID: "inquiry-feed", ExpiresAt: expiresAt,
					}, nil
				},
			)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			actual, err := s.PostInquiriesFeedStreamTicket(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "admin")),
				gen.PostInquiriesFeedStreamTicketRequestObject{},
			)

			require.NoError(t, err)
			response, ok := actual.(gen.PostInquiriesFeedStreamTicket201JSONResponse)
			require.True(t, ok)
			assert.Equal(t, "raw-feed-ticket", response.Ticket)
			assert.Equal(t, "inquiry-feed", response.StreamId)
			assert.Equal(t, expiresAt, response.ExpiresAt)
			assert.Equal(t, "operator-jane", capturedSubject)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証されていなければユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()
			uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.PostInquiriesFeedStreamTicket(
				context.Background(), gen.PostInquiriesFeedStreamTicketRequestObject{},
			)

			require.Error(t, err)
		})

		t.Run("管理者でない失敗をそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().IssueFeedTicket(gomock.Any(), gomock.Any()).
				Return(inquiryuc.TicketView{}, apperror.ErrPermissionDenied)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.PostInquiriesFeedStreamTicket(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "user")),
				gen.PostInquiriesFeedStreamTicketRequestObject{},
			)

			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})
	})
}
