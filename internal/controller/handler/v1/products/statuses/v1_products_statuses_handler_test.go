package productstatuses

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/products/statuses/gen"
	"go-boilerplate/internal/observability"
	statusuc "go-boilerplate/internal/usecase/product/status"
	mock_status "go-boilerplate/internal/usecase/product/status/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/products/statuses"

func newServer(t *testing.T) (*server, *mock_status.MockUsecase) {
	t.Helper()
	mockUC := mock_status.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}, mockUC
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockUC := mock_status.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockUC)

	testassert.AssertEchoRouterPath(t, targetPath, e.Router().Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Router().Routes())
}

func Test_server_GetProductStatuses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのDTO一覧を順序を保ってレスポンスへ詰め替える", func(t *testing.T) {
			t.Parallel()

			reviewingID, err := uuid.Parse("bdf44f06-227c-4549-b2c8-e57b32f06321")
			require.NoError(t, err)
			inStockID, err := uuid.Parse("093170fb-83a2-4864-a2b3-53236eaf3597")
			require.NoError(t, err)

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListStatuses(gomock.Any()).Return(statusuc.StatusDTOs{
				{ID: reviewingID, Code: 8, Name: "検討中"},
				{ID: inStockID, Code: 1, Name: "在庫あり"},
			}, nil)

			resp, err := s.GetProductStatuses(context.Background(), gen.GetProductStatusesRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductStatuses200JSONResponse)
			require.True(t, ok)

			assert.Equal(t, gen.GetProductStatuses200JSONResponse{
				{Id: reviewingID.ToPrimitive(), Code: 8, Name: "検討中"},
				{Id: inStockID.ToPrimitive(), Code: 1, Name: "在庫あり"},
			}, actual)
		})

		t.Run("空一覧の場合、空のレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListStatuses(gomock.Any()).Return(statusuc.StatusDTOs{}, nil)

			resp, err := s.GetProductStatuses(context.Background(), gen.GetProductStatusesRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductStatuses200JSONResponse)
			require.True(t, ok)
			assert.NotNil(t, actual)
			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListStatuses(gomock.Any()).Return(nil, apperror.ErrInternal)

			resp, err := s.GetProductStatuses(context.Background(), gen.GetProductStatusesRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
			require.Nil(t, resp)
		})
	})
}

func Test_toProductStatusResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステータスDTOの全項目をレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			id := uuidtestkit.NewTestFromSalt(t, "status_conv")
			actual := toProductStatusResponse(statusuc.StatusDTO{ID: id, Code: 8, Name: "検討中"})

			assert.Equal(t, id.ToPrimitive(), actual.Id)
			assert.Equal(t, 8, actual.Code)
			assert.Equal(t, "検討中", actual.Name)
		})
	})
}
