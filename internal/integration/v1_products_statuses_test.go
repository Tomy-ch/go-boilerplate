package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	productstatuseshandler "go-boilerplate/internal/controller/handler/v1/products/statuses"
	"go-boilerplate/internal/controller/handler/v1/products/statuses/gen"
	"go-boilerplate/internal/observability"
	statusuc "go-boilerplate/internal/usecase/product/status"
	mock_status "go-boilerplate/internal/usecase/product/status/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1ProductStatuses_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/product-statuses が未認証でも ProductStatusResponse 一覧を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			id, err := uuid.New()
			require.NoError(t, err)

			mockUC := mock_status.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListStatuses(gomock.Any()).Return(
				statusuc.StatusDTOs{{ID: id, Code: 8, Name: "検討中"}}, nil,
			)

			productstatuseshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/statuses", nil, nil)
			AssertJSONResponseType[[]gen.ProductStatusResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products/statuses が ErrInternal で 500 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_status.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListStatuses(gomock.Any()).Return(nil, apperror.ErrInternal)

			productstatuseshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/statuses", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
