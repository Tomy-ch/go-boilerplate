package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	prefectureshandler "go-boilerplate/internal/controller/handler/v1/prefectures"
	"go-boilerplate/internal/controller/handler/v1/prefectures/gen"
	"go-boilerplate/internal/observability"
	prefectureuc "go-boilerplate/internal/usecase/prefecture"
	mock_prefecture "go-boilerplate/internal/usecase/prefecture/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1Prefectures_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/prefectures が未認証でも PrefectureResponse 一覧を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			id, err := uuid.New()
			require.NoError(t, err)

			mockUC := mock_prefecture.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListPrefectures(gomock.Any()).Return(
				prefectureuc.PrefectureDTOs{{ID: id, Code: 1, Name: "都道府県"}}, nil,
			)

			prefectureshandler.BindHandler(e, tf, mockUC)

			// security: [] の公開エンドポイントのため、Authorization ヘッダー無しでも 200 が返る。
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/prefectures", nil, nil)
			AssertJSONResponseType[[]gen.PrefectureResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/prefectures が ErrInternal で 500 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_prefecture.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListPrefectures(gomock.Any()).Return(nil, apperror.ErrInternal)

			prefectureshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/prefectures", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
