package addresses

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/addresses/gen"
	"go-boilerplate/internal/observability"
	addressuc "go-boilerplate/internal/usecase/address"
	mock_address "go-boilerplate/internal/usecase/address/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/addresses"

func newServer(t *testing.T) (*server, *mock_address.MockUsecase) {
	t.Helper()
	mockUC := mock_address.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}, mockUC
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockUC := mock_address.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockUC)

	testassert.AssertEchoRouterPath(t, targetPath, e.Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Routes())
}

func Test_server_GetAddresses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ハイフン付き郵便番号を7桁へ正規化してusecaseへ渡す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			// ハイフンが除去された 7 桁が usecase へ渡ることを検証する。
			mockUC.EXPECT().
				LookupByPostalCode(gomock.Any(), "1000001").
				Return(&addressuc.Result{Candidates: []*addressuc.CandidateView{}, IsFallback: false}, nil)

			resp, err := s.GetAddresses(context.Background(), gen.GetAddressesRequestObject{
				Params: gen.GetAddressesParams{PostalCode: "100-0001"},
			})

			require.NoError(t, err)
			actual, ok := resp.(gen.GetAddresses200JSONResponse)
			require.True(t, ok)
			assert.False(t, actual.IsFallback)
			assert.Empty(t, actual.Candidates)
		})

		t.Run("候補を詰め替えprefecture_idの有無をnullableへ写像する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			s, mockUC := newServer(t)
			mockUC.EXPECT().
				LookupByPostalCode(gomock.Any(), "1000001").
				Return(&addressuc.Result{
					Candidates: []*addressuc.CandidateView{
						{PrefectureID: id.ToPtr(), PrefectureName: "東京都", City: "千代田区", Town: "千代田"},
						{PrefectureID: nil, PrefectureName: "不明県", City: "架空市", Town: "架空町"},
					},
					IsFallback: false,
				}, nil)

			resp, err := s.GetAddresses(context.Background(), gen.GetAddressesRequestObject{
				Params: gen.GetAddressesParams{PostalCode: "1000001"},
			})

			require.NoError(t, err)
			actual, ok := resp.(gen.GetAddresses200JSONResponse)
			require.True(t, ok)
			require.Len(t, actual.Candidates, 2)
			require.NotNil(t, actual.Candidates[0].PrefectureId)
			assert.Equal(t, id.ToPrimitive(), *actual.Candidates[0].PrefectureId)
			assert.Nil(t, actual.Candidates[1].PrefectureId)
			assert.Equal(t, "架空市", actual.Candidates[1].City)
		})

		t.Run("degrade時はis_fallback_trueと空候補をそのまま返す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().
				LookupByPostalCode(gomock.Any(), "1000001").
				Return(&addressuc.Result{Candidates: []*addressuc.CandidateView{}, IsFallback: true}, nil)

			resp, err := s.GetAddresses(context.Background(), gen.GetAddressesRequestObject{
				Params: gen.GetAddressesParams{PostalCode: "1000001"},
			})

			require.NoError(t, err)
			actual, ok := resp.(gen.GetAddresses200JSONResponse)
			require.True(t, ok)
			assert.True(t, actual.IsFallback)
			assert.Empty(t, actual.Candidates)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().
				LookupByPostalCode(gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrInternal)

			_, err := s.GetAddresses(context.Background(), gen.GetAddressesRequestObject{
				Params: gen.GetAddressesParams{PostalCode: "1000001"},
			})

			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_toAddressCandidates(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("候補一覧をレスポンスDTOへ変換する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			actual := toAddressCandidates([]*addressuc.CandidateView{
				{PrefectureID: id.ToPtr(), PrefectureName: "東京都", City: "千代田区", Town: "千代田"},
				{PrefectureID: nil, PrefectureName: "不明県", City: "架空市", Town: "架空町"},
			})

			require.Len(t, actual, 2)
			require.NotNil(t, actual[0].PrefectureId)
			assert.Equal(t, id.ToPrimitive(), *actual[0].PrefectureId)
			assert.Equal(t, "東京都", actual[0].PrefectureName)
			assert.Nil(t, actual[1].PrefectureId)
		})

		t.Run("空スライスは空スライスを返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, toAddressCandidates([]*addressuc.CandidateView{}))
		})
	})
}

func Test_toNullableUUID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのIDをプリミティブ型へ変換する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			actual := toNullableUUID(id.ToPtr())

			require.NotNil(t, actual)
			assert.Equal(t, id.ToPrimitive(), *actual)
		})

		t.Run("nilならnilを返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, toNullableUUID(nil))
		})
	})
}
