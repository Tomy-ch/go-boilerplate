package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	feedticket "go-boilerplate/internal/controller/handler/v1/inquiries/feed/streamticket"
	"go-boilerplate/internal/controller/handler/v1/inquiries/feed/streamticket/gen"
	"go-boilerplate/internal/observability"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	mock_inquiryuc "go-boilerplate/internal/usecase/inquiry/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

const inquiryFeedStreamTicketPath = "/v1/inquiries/feed/stream-ticket"

func TestV1InquiriesFeedStreamTicket_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("発行は201でticketとstreamIdを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().IssueFeedTicket(gomock.Any(), gomock.Any()).Return(inquiryuc.TicketView{
				Ticket:    "1pYl2d6u9Zc2zr4bC8qRxS3tU5vW7xY9",
				StreamID:  "inquiry-feed",
				ExpiresAt: time.Date(2026, time.September, 1, 10, 5, 0, 0, time.UTC),
			}, nil)

			feedticket.BindHandler(e, observability.NewNoopTracerFactory(t), uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_admin"))
			actual := StartServer(t, e).DoJSON(http.MethodPost, inquiryFeedStreamTicketPath, nil, headers)

			require.Equal(t, http.StatusCreated, actual.StatusCode)
			AssertJSONResponseType[gen.InquiryStreamTicketResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者以外の発行は403を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().IssueFeedTicket(gomock.Any(), gomock.Any()).
				Return(inquiryuc.TicketView{}, apperror.ErrPermissionDenied)

			feedticket.BindHandler(e, observability.NewNoopTracerFactory(t), uc)
			UseAppErrorHandler(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_user"))
			actual := StartServer(t, e).DoJSON(http.MethodPost, inquiryFeedStreamTicketPath, nil, headers)

			AssertErrorResponse(t, actual, http.StatusForbidden)
		})
	})
}
