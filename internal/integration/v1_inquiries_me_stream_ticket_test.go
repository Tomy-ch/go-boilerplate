package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	streamticket "go-boilerplate/internal/controller/handler/v1/inquiries/me/streamticket"
	"go-boilerplate/internal/controller/handler/v1/inquiries/me/streamticket/gen"
	"go-boilerplate/internal/observability"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	mock_inquiryuc "go-boilerplate/internal/usecase/inquiry/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

const meInquiryStreamTicketPath = "/v1/inquiries/me/stream-ticket"

func TestV1InquiriesMeStreamTicket_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("発行は201でticketとstreamIdを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().IssueStreamTicket(gomock.Any(), gomock.Any()).Return(inquiryuc.TicketView{
				Ticket:    "0oXk1c5t8Yb1yq3aB7pQwR2sT4uV6wX8",
				StreamID:  uuidtestkit.NewTestFromSalt(t, "int_inq").String(),
				ExpiresAt: time.Date(2026, time.September, 1, 10, 5, 0, 0, time.UTC),
			}, nil)

			streamticket.BindHandler(e, observability.NewNoopTracerFactory(t), uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_inq_user"))
			actual := StartServer(t, e).DoJSON(http.MethodPost, meInquiryStreamTicketPath, nil, headers)

			require.Equal(t, http.StatusCreated, actual.StatusCode)
			AssertJSONResponseType[gen.InquiryStreamTicketResponse](t, actual)
			assert.NotContains(t, actual.Header.Get("Location"), "0oXk1c5t8Yb1yq3aB7pQwR2sT4uV6wX8")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購読する問い合わせが無ければ404を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().IssueStreamTicket(gomock.Any(), gomock.Any()).
				Return(inquiryuc.TicketView{}, apperror.ErrNotFound)

			streamticket.BindHandler(e, observability.NewNoopTracerFactory(t), uc)
			UseAppErrorHandler(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_inq_user"))
			actual := StartServer(t, e).DoJSON(http.MethodPost, meInquiryStreamTicketPath, nil, headers)

			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("未認証の発行は401を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))
			streamticket.BindHandler(e, observability.NewNoopTracerFactory(t), uc)
			UseAppErrorHandler(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodPost, meInquiryStreamTicketPath, nil, nil)

			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})
	})
}
