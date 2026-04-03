package integration

import (
	"net/http"
	"testing"
	"time"

	searchhandler "boilerplate-go/internal/controller/handler/v1/users/search"
	"boilerplate-go/internal/controller/handler/v1/users/search/gen"
	"boilerplate-go/internal/observability"
	mock_search "boilerplate-go/internal/usecase/user/search/mock"
	"boilerplate-go/internal/usecase/user/search/query"

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

	t.Run("GET /v1/users/searchのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e := echo.New()
		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)

		mockApp := mock_search.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			CountUsersByKeyword(gomock.Any(), gomock.Any()).
			Return(int64(1), nil)
		mockApp.EXPECT().
			ListUsersByKeyword(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(expectedDTO, nil)

		searchhandler.BindHandler(e, tf, mockApp)

		actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users/search", nil, nil)
		AssertJSONResponse(t, gen.UsersSearchResponse{}, actual)
	})
}
