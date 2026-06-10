package integration

import (
	"net/http"
	"testing"
	"time"

	searchhandler "go-boilerplate/internal/controller/handler/v1/users/search"
	"go-boilerplate/internal/controller/handler/v1/users/search/gen"
	"go-boilerplate/internal/observability"
	usecase_search "go-boilerplate/internal/usecase/user/search"
	mock_search "go-boilerplate/internal/usecase/user/search/mock"
	"go-boilerplate/internal/usecase/user/search/query"

	"github.com/labstack/echo/v4"
	"go.uber.org/mock/gomock"
)

func TestV1UsersSearch_Integration(t *testing.T) {
	t.Parallel()

	expectedDTO := query.UserSearchResults{
		&query.UserSearchResult{
			FirstName:      "User1",
			LastName:       "One",
			Email:          "user1@example.com",
			Phone:          "1234567890",
			PostalCode:     "123-4567",
			PrefectureName: "Tokyo",
			City:           "Chiyoda",
			Street:         "1-1",
			Building:       nil,
			RegisteredAt:   time.Now().UTC(),
			DeletedAt:      nil,
		},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/users/searchがUsersSearchResponseを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_search.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersByKeywordWithTotal(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&usecase_search.UserSearchListView{Items: expectedDTO, Total: 1}, nil)

			searchhandler.BindHandler(e, tf, mockApp)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users/search", nil, nil)
			AssertJSONResponse(t, gen.UsersSearchResponse{}, actual)
		})
	})
}
