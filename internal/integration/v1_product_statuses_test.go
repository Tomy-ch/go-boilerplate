package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	productstatuseshandler "go-boilerplate/internal/controller/handler/v1/product-statuses"
	"go-boilerplate/internal/controller/handler/v1/product-statuses/gen"
	"go-boilerplate/internal/observability"
	productstatusuc "go-boilerplate/internal/usecase/productstatus"
	mock_productstatus "go-boilerplate/internal/usecase/productstatus/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
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

			mockUC := mock_productstatus.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProductStatuses(gomock.Any()).Return(
				productstatusuc.ProductStatusDTOs{{ID: id, Code: 8, Name: "検討中", SortKey: 1}}, nil,
			)

			productstatuseshandler.BindHandler(e, tf, mockUC)

			// security: [] の公開エンドポイントのため、Authorization ヘッダー無しでも 200 が返る。
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/product-statuses", nil, nil)
			AssertJSONResponseType[[]gen.ProductStatusResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/product-statuses が ErrInternal で 500 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_productstatus.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProductStatuses(gomock.Any()).Return(nil, apperror.ErrInternal)

			productstatuseshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/product-statuses", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
