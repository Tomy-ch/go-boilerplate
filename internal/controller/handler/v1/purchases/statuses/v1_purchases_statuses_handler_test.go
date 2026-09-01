package purchasestatuses

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/purchases/statuses/gen"
	"go-boilerplate/internal/observability"
	statusuc "go-boilerplate/internal/usecase/purchase/status"
	mock_status "go-boilerplate/internal/usecase/purchase/status/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/purchases/statuses"

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

func Test_server_GetPurchaseStatuses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのDTO一覧を順序を保ってレスポンスへ詰め替える", func(t *testing.T) {
			t.Parallel()

			canceledID, err := uuid.Parse("e9d72547-adfe-48d9-9037-bd1f55d4158b")
			require.NoError(t, err)
			paidID, err := uuid.Parse("4b8f0e2a-1c3d-4a5e-8b7f-2d9c0e1a3b4c")
			require.NoError(t, err)

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListStatuses(gomock.Any()).Return(statusuc.StatusDTOs{
				{ID: canceledID, Code: 6, Name: "キャンセル"},
				{ID: paidID, Code: 7, Name: "支払い済み"},
			}, nil)

			resp, err := s.GetPurchaseStatuses(context.Background(), gen.GetPurchaseStatusesRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetPurchaseStatuses200JSONResponse)
			require.True(t, ok)

			assert.Equal(t, gen.GetPurchaseStatuses200JSONResponse{
				{Id: canceledID.ToPrimitive(), Code: 6, Name: "キャンセル"},
				{Id: paidID.ToPrimitive(), Code: 7, Name: "支払い済み"},
			}, actual)
		})

		t.Run("空一覧の場合、空のレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListStatuses(gomock.Any()).Return(statusuc.StatusDTOs{}, nil)

			resp, err := s.GetPurchaseStatuses(context.Background(), gen.GetPurchaseStatusesRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetPurchaseStatuses200JSONResponse)
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

			resp, err := s.GetPurchaseStatuses(context.Background(), gen.GetPurchaseStatusesRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Nil(t, resp)
		})
	})
}

func Test_toPurchaseStatusRef(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステータスDTOの全項目をレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			id := uuidtestkit.NewTestFromSalt(t, "status_conv")
			actual := toPurchaseStatusRef(statusuc.StatusDTO{ID: id, Code: 7, Name: "支払い済み"})

			assert.Equal(t, id.ToPrimitive(), actual.Id)
			assert.Equal(t, int64(7), actual.Code)
			assert.Equal(t, "支払い済み", actual.Name)
		})
	})
}
