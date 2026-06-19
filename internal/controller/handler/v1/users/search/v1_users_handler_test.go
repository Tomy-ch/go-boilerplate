package search

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	gen "go-boilerplate/internal/controller/handler/v1/users/search/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	usecase_search "go-boilerplate/internal/usecase/user/search"
	mock_query "go-boilerplate/internal/usecase/user/search/mock"
	"go-boilerplate/internal/usecase/user/search/query"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users/search"

func newServer(t *testing.T) (*server, *mock_query.MockUsecase) {
	t.Helper()
	mockApp := mock_query.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockApp}, mockApp
}

// wantSearchItem は、本番の変換とは独立な検証用オラクル（フィールド取り違え検出）。
func wantSearchItem(r *query.UserSearchResult) gen.UsersSearchResponseItem {
	return gen.UsersSearchResponseItem{
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

	testassert.AssertEchoRouterPath(t, targetPath, e.Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Routes())
}

func Test_server_GetUsersSearch(t *testing.T) {
	t.Parallel()

	expectedPage := 1
	expectedPerPage := 10

	t1 := time.Now().UTC()
	t2 := t1.Add(time.Hour)

	mockPaging, err := paging.NewPagingFrom1Based(new(expectedPage), new(expectedPerPage))
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

		exec := func(t *testing.T, dtos query.UserSearchResults, total int64) {
			t.Helper()

			wantUsers := make([]gen.UsersSearchResponseItem, len(dtos))
			for i, d := range dtos {
				wantUsers[i] = wantSearchItem(d)
			}
			expectedResponse := gen.UsersSearchResponse{
				Users:  wantUsers,
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
				Total:  total,
			}

			var gotFilter *usecase_search.SearchParams
			s, mockApp := newServer(t)
			mockApp.EXPECT().ListUsersByKeywordWithTotal(gomock.Any(), gomock.Any(), mockPaging).
				DoAndReturn(func(_ context.Context, f *usecase_search.SearchParams, _ *paging.Paging) (*usecase_search.UserSearchListView, error) {
					gotFilter = f
					return &usecase_search.UserSearchListView{Items: dtos, Total: total}, nil
				})

			resp, err := s.GetUsersSearch(context.Background(), mockParams)
			require.NoError(t, err)

			assert.Equal(t, wantFilter, gotFilter)

			actual, ok := resp.(gen.GetUsersSearch200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, expectedResponse, gen.UsersSearchResponse(actual))
		}

		t.Run("複数ユーザーが存在する場合、検索結果が返る", func(t *testing.T) {
			t.Parallel()
			dtos := query.UserSearchResults{
				&query.UserSearchResult{
					FirstName: "F1", LastName: "L1", Email: "u1@example.com", Phone: "090-0000-0001",
					PostalCode: "123-0001", PrefectureName: "Tokyo", City: "Shibuya", Street: "1-1-1",
					Building: new("B1"), RegisteredAt: t1,
				},
				&query.UserSearchResult{
					FirstName: "F2", LastName: "L2", Email: "u2@example.com", Phone: "090-0000-0002",
					PostalCode: "123-0002", PrefectureName: "Osaka", City: "Kita", Street: "2-2-2",
					RegisteredAt: t2,
				},
			}
			exec(t, dtos, 2)
		})

		t.Run("0件の場合、空の検索結果が返る", func(t *testing.T) {
			t.Parallel()
			exec(t, query.UserSearchResults{}, 0)
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
			resp, err := s.GetUsersSearch(context.Background(), invalidParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("ListUsersByKeywordWithTotalがエラーを返す場合、エラーが返る", func(t *testing.T) {
			t.Parallel()

			s, mockApp := newServer(t)
			mockApp.EXPECT().ListUsersByKeywordWithTotal(gomock.Any(), gomock.Any(), mockPaging).
				Return(nil, apperror.ErrInternal)

			resp, err := s.GetUsersSearch(context.Background(), mockParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
