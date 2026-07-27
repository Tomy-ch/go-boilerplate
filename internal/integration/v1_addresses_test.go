package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	addresseshandler "go-boilerplate/internal/controller/handler/v1/addresses"
	"go-boilerplate/internal/controller/handler/v1/addresses/gen"
	"go-boilerplate/internal/observability"
	addressuc "go-boilerplate/internal/usecase/address"
	mock_address "go-boilerplate/internal/usecase/address/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"go.uber.org/mock/gomock"
)

func TestV1AddressesIntegration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/addresses が候補を含む AddressCandidatesResponse を返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			if err != nil {
				t.Fatal(err)
			}

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_address.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				LookupByPostalCode(gomock.Any(), "1000001").
				Return(&addressuc.Result{
					Candidates: []*addressuc.CandidateView{
						{PrefectureID: id.ToPtr(), PrefectureName: "東京都", City: "千代田区", Town: "千代田"},
					},
					IsFallback: false,
				}, nil)

			addresseshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/addresses?postalCode=100-0001", nil, nil,
			)
			AssertJSONResponseType[gen.AddressCandidatesResponse](t, actual)
		})

		t.Run("該当なしでも 200 で空 candidates_is_fallback_false を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_address.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				LookupByPostalCode(gomock.Any(), gomock.Any()).
				Return(&addressuc.Result{Candidates: []*addressuc.CandidateView{}, IsFallback: false}, nil)

			addresseshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/addresses?postalCode=0000000", nil, nil,
			)
			AssertJSONResponseType[gen.AddressCandidatesResponse](t, actual)
		})

		t.Run("degrade: 外部lookup障害でも 503 でなく 200 で AddressCandidatesResponse を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			// 外部 lookup 障害を usecase の degrade（error を返さず IsFallback:true）として模擬する。
			// HTTP 境界では「503 でなく 200 へ倒れる（登録を止めない）」ことを検証し、is_fallback 値の
			// 正しさは controller 単体テストが担保する。
			mockUC := mock_address.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				LookupByPostalCode(gomock.Any(), gomock.Any()).
				Return(&addressuc.Result{Candidates: []*addressuc.CandidateView{}, IsFallback: true}, nil)

			addresseshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/addresses?postalCode=1000001", nil, nil,
			)
			AssertJSONResponseType[gen.AddressCandidatesResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("degrade対象外のエラー（県名解決のDB障害等）は 500 へ写像する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			// 外部 lookup 障害の degrade とは異なり、prefecture 解決の非NotFoundエラーは
			// usecase がそのまま伝播する。実配線の errorhandler で 500 へ写像されることを検証する。
			mockUC := mock_address.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				LookupByPostalCode(gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrInternal)

			addresseshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/addresses?postalCode=1000001", nil, nil,
			)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
