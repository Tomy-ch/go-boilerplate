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

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
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

		t.Run("DELETE /v1/prefectures が Allow ヘッダー付きの 405 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			// 本番の 405 は Echo のルータではなく OpenAPI バリデーションミドルウェアが送出するため、
			// 実経路を再現するにはこのミドルウェアの配線が要る。
			// Allow の情報源は Echo のルータ側になる（/v1/prefectures に重なる可変パスが無く、
			// Echo 自身も 405 と判断して ContextKeyHeaderAllow を解決するため）。
			// spec 側を情報源とする経路は TestV1UsersMe_Integration が担う。
			useOpenAPIValidation(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			prefectureshandler.BindHandler(e, tf, mock_prefecture.NewMockUsecase(ctrl))

			actual := StartServer(t, e).DoJSON(http.MethodDelete, "/v1/prefectures", nil, nil)
			AssertErrorResponse(t, actual, http.StatusMethodNotAllowed)
			assert.Equal(t, "OPTIONS, GET", actual.Header.Get(echo.HeaderAllow))
		})
	})
}
