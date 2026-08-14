package search

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/testkit/testauth"
	gen "go-boilerplate/internal/controller/handler/v1/users/search/gen"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/tools/paging"
	usecase_search "go-boilerplate/internal/usecase/user/search"
	mock_query "go-boilerplate/internal/usecase/user/search/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users/search"

// searchSubject は、検索テストで使う認証主体の subject です。
const searchSubject = "11111111-1111-1111-1111-111111111111"

func newServer(t *testing.T) (*server, *mock_query.MockUsecase) {
	t.Helper()
	mockApp := mock_query.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockApp}, mockApp
}

// wantSearchItem は、本番の変換とは独立な検証用オラクル（フィールド取り違え検出）。
func wantSearchItem(r *usecase_search.UserSearchResult) gen.UsersSearchResponseItem {
	return gen.UsersSearchResponseItem{
		Id:           r.ID.ToPrimitive(),
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		Email:        types.Email(r.Email),
		Phone:        r.Phone,
		PostalCode:   r.PostalCode,
		Prefecture:   r.PrefectureName,
		City:         r.City,
		Street:       r.Street,
		Building:     r.Building,
		RegisteredAt: r.RegisteredAt,
		DeletedAt:    r.DeletedAt,
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockApp := mock_query.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockApp)

	testassert.AssertEchoRouterPath(t, targetPath, e.Router().Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Router().Routes())
}

func Test_server_GetUsersSearch(t *testing.T) {
	t.Parallel()

	expectedPage := 1
	expectedPerPage := 10

	t1 := time.Now().UTC()
	t2 := t1.Add(time.Hour)

	mockPage, err := paging.NewPageFrom1Based(new(expectedPage), new(expectedPerPage))
	require.NoError(t, err)

	// Keyword/Active を設定し、ハンドラの filter 詰め替えを検証可能にする。
	mockParams := gen.GetUsersSearchRequestObject{
		Params: gen.GetUsersSearchParams{
			Page:    new(expectedPage),
			PerPage: new(expectedPerPage),
			Keyword: new("alice"),
			Active:  new(true),
		},
	}
	wantFilter := &usecase_search.SearchParams{Keyword: new("alice"), Active: new(true)}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		exec := func(t *testing.T, dtos usecase_search.UserSearchResults, total int64) {
			t.Helper()

			wantUsers := make([]gen.UsersSearchResponseItem, len(dtos))
			for i, d := range dtos {
				wantUsers[i] = wantSearchItem(d)
			}
			expectedResponse := gen.UsersSearchResponse{
				Users:  wantUsers,
				Limit:  mockPage.Limit(),
				Offset: mockPage.Offset(),
				Total:  total,
			}

			var gotFilter *usecase_search.SearchParams
			s, mockApp := newServer(t)
			mockApp.EXPECT().ListUsersByKeywordWithTotal(gomock.Any(), gomock.Any(), gomock.Any(), mockPage).
				DoAndReturn(func(_ context.Context, _ *authbd.Authn, f *usecase_search.SearchParams, _ *paging.Page) (*usecase_search.UserSearchListView, error) {
					gotFilter = f
					return &usecase_search.UserSearchListView{Items: dtos, Total: total}, nil
				})

			resp, err := s.GetUsersSearch(testauth.MakeAvailableAuthn(context.Background(), t, searchSubject), mockParams)
			require.NoError(t, err)

			assert.Equal(t, wantFilter, gotFilter)

			actual, ok := resp.(gen.GetUsersSearch200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, expectedResponse, gen.UsersSearchResponse(actual))
		}

		t.Run("複数ユーザーが存在する場合、検索結果が返る", func(t *testing.T) {
			t.Parallel()
			dtos := usecase_search.UserSearchResults{
				&usecase_search.UserSearchResult{
					ID:        uuidtestkit.NewTestFromSalt(t, "user_search_result_1"),
					FirstName: "F1", LastName: "L1", Email: "u1@example.com", Phone: "090-0000-0001",
					PostalCode: "123-0001", PrefectureName: "Tokyo", City: "Shibuya", Street: "1-1-1",
					Building: new("B1"), RegisteredAt: t1,
				},
				&usecase_search.UserSearchResult{
					ID:        uuidtestkit.NewTestFromSalt(t, "user_search_result_2"),
					FirstName: "F2", LastName: "L2", Email: "u2@example.com", Phone: "090-0000-0002",
					PostalCode: "123-0002", PrefectureName: "Osaka", City: "Kita", Street: "2-2-2",
					RegisteredAt: t2,
				},
			}
			exec(t, dtos, 2)
		})

		t.Run("0件の場合、空の検索結果が返る", func(t *testing.T) {
			t.Parallel()
			exec(t, usecase_search.UserSearchResults{}, 0)
		})

		t.Run("Keyword/Activeが未指定の場合、空のSearchParamsで呼び出される", func(t *testing.T) {
			t.Parallel()

			noFilterParams := gen.GetUsersSearchRequestObject{
				Params: gen.GetUsersSearchParams{
					Page:    new(expectedPage),
					PerPage: new(expectedPerPage),
				},
			}

			var gotFilter *usecase_search.SearchParams
			s, mockApp := newServer(t)
			mockApp.EXPECT().ListUsersByKeywordWithTotal(gomock.Any(), gomock.Any(), gomock.Any(), mockPage).
				DoAndReturn(func(_ context.Context, _ *authbd.Authn, f *usecase_search.SearchParams, _ *paging.Page) (*usecase_search.UserSearchListView, error) {
					gotFilter = f
					return &usecase_search.UserSearchListView{Items: usecase_search.UserSearchResults{}, Total: 0}, nil
				})

			resp, err := s.GetUsersSearch(testauth.MakeAvailableAuthn(context.Background(), t, searchSubject), noFilterParams)
			require.NoError(t, err)

			// フィルタ未指定でも nil ではなく Keyword/Active が共に nil の空 SearchParams が渡る。
			assert.Equal(t, &usecase_search.SearchParams{}, gotFilter)

			_, ok := resp.(gen.GetUsersSearch200JSONResponse)
			require.True(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ページング処理が失敗した場合、ErrInvalidArgumentが返る", func(t *testing.T) {
			t.Parallel()

			invalidParams := gen.GetUsersSearchRequestObject{
				Params: gen.GetUsersSearchParams{
					Page:    new(1_000_000), // paging.maxPage 超過
					PerPage: new(expectedPerPage),
				},
			}

			s, _ := newServer(t)
			resp, err := s.GetUsersSearch(testauth.MakeAvailableAuthn(context.Background(), t, searchSubject), invalidParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("ListUsersByKeywordWithTotalがエラーを返す場合、エラーが返る", func(t *testing.T) {
			t.Parallel()

			s, mockApp := newServer(t)
			mockApp.EXPECT().ListUsersByKeywordWithTotal(gomock.Any(), gomock.Any(), gomock.Any(), mockPage).
				Return(nil, apperror.ErrInternal)

			resp, err := s.GetUsersSearch(testauth.MakeAvailableAuthn(context.Background(), t, searchSubject), mockParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("未認証の場合_ErrUnauthenticatedUser", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.GetUsersSearch(context.Background(), mockParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})
	})
}
