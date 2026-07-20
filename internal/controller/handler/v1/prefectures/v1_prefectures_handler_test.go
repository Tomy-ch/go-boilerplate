package prefectures

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/prefectures/gen"
	"go-boilerplate/internal/observability"
	prefectureuc "go-boilerplate/internal/usecase/prefecture"
	mock_prefecture "go-boilerplate/internal/usecase/prefecture/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/prefectures"

func newServer(t *testing.T) (*server, *mock_prefecture.MockUsecase) {
	t.Helper()
	mockUC := mock_prefecture.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}, mockUC
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockUC := mock_prefecture.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockUC)

	testassert.AssertEchoRouterPath(t, targetPath, e.Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Routes())
}

func Test_server_GetPrefectures(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのDTO一覧をcode昇順のレスポンスへ詰め替える", func(t *testing.T) {
			t.Parallel()

			tokyoID, err := uuid.Parse("101caa1e-84e7-4ceb-9108-50d40b6be1a3")
			require.NoError(t, err)
			osakaID, err := uuid.Parse("d647fc85-ff46-4530-88cb-198f4a68a9d7")
			require.NoError(t, err)

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListPrefectures(gomock.Any()).Return(prefectureuc.PrefectureDTOs{
				{ID: tokyoID, Code: 13, Name: "東京都"},
				{ID: osakaID, Code: 27, Name: "大阪府"},
			}, nil)

			resp, err := s.GetPrefectures(context.Background(), gen.GetPrefecturesRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetPrefectures200JSONResponse)
			require.True(t, ok)

			assert.Equal(t, gen.GetPrefectures200JSONResponse{
				{Id: tokyoID.ToPrimitive(), Code: 13, Name: "東京都"},
				{Id: osakaID.ToPrimitive(), Code: 27, Name: "大阪府"},
			}, actual)
		})

		t.Run("空一覧の場合、空のレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListPrefectures(gomock.Any()).Return(prefectureuc.PrefectureDTOs{}, nil)

			resp, err := s.GetPrefectures(context.Background(), gen.GetPrefecturesRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetPrefectures200JSONResponse)
			require.True(t, ok)
			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListPrefectures(gomock.Any()).Return(nil, apperror.ErrInternal)

			_, err := s.GetPrefectures(context.Background(), gen.GetPrefecturesRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
