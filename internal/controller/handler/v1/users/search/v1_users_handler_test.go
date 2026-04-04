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
	"go-boilerplate/pkg/ptr"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users/search"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	mockApp := mock_query.NewMockUsecase(ctrl)

	BindHandler(e, tf, mockApp)

	expectedMethods := []string{
		http.MethodGet,
	}

	testassert.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	testassert.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func Test_server_GetUsersSearch(t *testing.T) {
	t.Parallel()

	expectedPage := 1
	expectedPerPage := 10
	expectedTotal := int64(2)

	t1 := time.Now().UTC()
	t2 := t1.Add(time.Hour)

	mockPaging, err := paging.NewPagingFrom1Based(ptr.To(expectedPage), ptr.To(expectedPerPage))
	require.NoError(t, err)

	mockParams := gen.GetUsersSearchRequestObject{
		Params: gen.GetUsersSearchParams{
			Page:    ptr.To(expectedPage),
			PerPage: ptr.To(expectedPerPage),
		},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数ユーザーが存在する場合、検索結果が返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

			expectedResponse := gen.UsersSearchResponse{
				Users: []gen.UsersSearchResponseItem{
					{
						FirstName:    "F1",
						LastName:     "L1",
						Email:        types.Email("u1@example.com"),
						Phone:        "090-0000-0001",
						PostalCode:   "123-0001",
						Prefecture:   "Tokyo",
						City:         "Shibuya",
						Street:       "1-1-1",
						Building:     ptr.To("B1"),
						RegisteredAt: t1,
						DeletedAt:    nil,
					},
					{
						FirstName:    "F2",
						LastName:     "L2",
						Email:        types.Email("u2@example.com"),
						Phone:        "090-0000-0002",
						PostalCode:   "123-0002",
						Prefecture:   "Osaka",
						City:         "Kita",
						Street:       "2-2-2",
						Building:     nil,
						RegisteredAt: t2,
						DeletedAt:    nil,
					},
				},
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
				Total:  expectedTotal,
			}

			mockDTO := query.UserSearchResults{
				&query.UserSearchResult{
					FirstName:      "F1",
					LastName:       "L1",
					Email:          "u1@example.com",
					Phone:          "090-0000-0001",
					PostalCode:     "123-0001",
					PrefectureName: "Tokyo",
					City:           "Shibuya",
					Street:         "1-1-1",
					Building:       ptr.To("B1"),
					RegisteredAt:   t1,
					DeletedAt:      nil,
				},
				&query.UserSearchResult{
					FirstName:      "F2",
					LastName:       "L2",
					Email:          "u2@example.com",
					Phone:          "090-0000-0002",
					PostalCode:     "123-0002",
					PrefectureName: "Osaka",
					City:           "Kita",
					Street:         "2-2-2",
					Building:       nil,
					RegisteredAt:   t2,
					DeletedAt:      nil,
				},
			}

			mockApp := mock_query.NewMockUsecase(ctrl)
			mockApp.EXPECT().ListUsersByKeyword(
				gomock.Any(), gomock.AssignableToTypeOf(&usecase_search.SearchParams{}), mockPaging,
			).Return(mockDTO, nil)
			mockApp.EXPECT().CountUsersByKeyword(
				gomock.Any(), gomock.AssignableToTypeOf(&usecase_search.SearchParams{}),
			).Return(expectedTotal, nil)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsersSearch(ctx, mockParams)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsersSearch200JSONResponse)
			require.True(t, ok)

			require.Equal(t, expectedResponse, gen.UsersSearchResponse(actual))
		})

		t.Run("単一ユーザーが存在する場合、検索結果が返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

			expectedResponse := gen.UsersSearchResponse{
				Users: []gen.UsersSearchResponseItem{
					{
						FirstName:    "F1",
						LastName:     "L1",
						Email:        types.Email("u1@example.com"),
						Phone:        "090-0000-0001",
						PostalCode:   "123-0001",
						Prefecture:   "Tokyo",
						City:         "Shibuya",
						Street:       "1-1-1",
						Building:     ptr.To("B1"),
						RegisteredAt: t1,
						DeletedAt:    nil,
					},
				},
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
				Total:  expectedTotal,
			}

			mockDTO := query.UserSearchResults{
				&query.UserSearchResult{
					FirstName:      "F1",
					LastName:       "L1",
					Email:          "u1@example.com",
					Phone:          "090-0000-0001",
					PostalCode:     "123-0001",
					PrefectureName: "Tokyo",
					City:           "Shibuya",
					Street:         "1-1-1",
					Building:       ptr.To("B1"),
					RegisteredAt:   t1,
					DeletedAt:      nil,
				},
			}

			mockApp := mock_query.NewMockUsecase(ctrl)
			mockApp.EXPECT().ListUsersByKeyword(
				gomock.Any(), gomock.AssignableToTypeOf(&usecase_search.SearchParams{}), mockPaging,
			).Return(mockDTO, nil)
			mockApp.EXPECT().CountUsersByKeyword(
				gomock.Any(), gomock.AssignableToTypeOf(&usecase_search.SearchParams{}),
			).Return(expectedTotal, nil)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsersSearch(ctx, mockParams)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsersSearch200JSONResponse)
			require.True(t, ok)

			require.Equal(t, expectedResponse, gen.UsersSearchResponse(actual))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ページング処理が失敗した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

			invalidPage := 1_000_000
			invalidParams := gen.GetUsersSearchRequestObject{
				Params: gen.GetUsersSearchParams{
					Page:    ptr.To(invalidPage),
					PerPage: ptr.To(expectedPerPage),
				},
			}

			mockApp := mock_query.NewMockUsecase(ctrl)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsersSearch(ctx, invalidParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("Usecaseがエラーを返す場合、エラーが返る", func(t *testing.T) {
			t.Parallel()

			t.Run("ListUsersByKeywordがエラーを返す場合", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				ctrl := gomock.NewController(t)
				lt := observability.NewMockControllerLayerTracer(t)

				expectedErr := apperror.ErrInternal

				mockApp := mock_query.NewMockUsecase(ctrl)
				mockApp.EXPECT().ListUsersByKeyword(
					gomock.Any(), gomock.AssignableToTypeOf(&usecase_search.SearchParams{}), mockPaging,
				).Return(nil, expectedErr)

				s := &server{tracer: lt, uc: mockApp}
				resp, err := s.GetUsersSearch(ctx, mockParams)
				require.Nil(t, resp)
				require.ErrorIs(t, err, expectedErr)
			})

			t.Run("CountUsersByKeywordがエラーを返す場合", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				ctrl := gomock.NewController(t)
				lt := observability.NewMockControllerLayerTracer(t)

				expectedErr := apperror.ErrInternal

				mockDTO := query.UserSearchResults{
					&query.UserSearchResult{FirstName: "F1"},
				}
				mockApp := mock_query.NewMockUsecase(ctrl)
				mockApp.EXPECT().ListUsersByKeyword(
					gomock.Any(), gomock.AssignableToTypeOf(&usecase_search.SearchParams{}), mockPaging,
				).Return(mockDTO, nil)
				mockApp.EXPECT().CountUsersByKeyword(
					gomock.Any(), gomock.AssignableToTypeOf(&usecase_search.SearchParams{}),
				).Return(int64(0), expectedErr)

				s := &server{tracer: lt, uc: mockApp}
				resp, err := s.GetUsersSearch(ctx, mockParams)
				require.Nil(t, resp)
				require.ErrorIs(t, err, expectedErr)
			})
		})
	})
}
