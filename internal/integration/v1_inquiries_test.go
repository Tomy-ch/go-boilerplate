package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	inquiries "go-boilerplate/internal/controller/handler/v1/inquiries"
	"go-boilerplate/internal/controller/handler/v1/inquiries/gen"
	"go-boilerplate/internal/observability"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	mock_inquiryuc "go-boilerplate/internal/usecase/inquiry/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

const inquiriesPath = "/v1/inquiries"

func TestV1Inquiries_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧は200で要約の配列を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().ListInquiries(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&inquiryuc.InquiryListView{Items: []inquiryuc.InquirySummaryView{{
					ID:        uuidtestkit.NewTestFromSalt(t, "int_inq"),
					UserID:    uuidtestkit.NewTestFromSalt(t, "int_inq_user"),
					CreatedAt: time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, time.September, 1, 11, 0, 0, 0, time.UTC),
				}}}, nil)

			inquiries.BindHandler(e, observability.NewNoopTracerFactory(t), uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_admin"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, inquiriesPath, nil, headers)

			require.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.InquiryListResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者以外の一覧取得は403を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().ListInquiries(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrPermissionDenied)

			inquiries.BindHandler(e, observability.NewNoopTracerFactory(t), uc)
			UseAppErrorHandler(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_user"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, inquiriesPath, nil, headers)

			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("未認証の一覧取得は401を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))
			inquiries.BindHandler(e, observability.NewNoopTracerFactory(t), uc)
			UseAppErrorHandler(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, inquiriesPath, nil, nil)

			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})
	})
}
