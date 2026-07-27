package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/v1/users/feed"
	"go-boilerplate/internal/controller/handler/v1/users/feed/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestV1UsersFeed_Integration(t *testing.T) {
	t.Parallel()

	expectedDTO := user.UserView{FirstName: "Feed1", LastName: "One", Email: "feed1@example.com", Phone: "1234567890"}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/users/feedがフィードを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			nextCursor := "next-opaque-cursor"
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersFeed(gomock.Any(), gomock.Any()).
				Return(&user.UserFeedView{Items: []user.UserView{expectedDTO}, NextCursor: &nextCursor}, nil)

			feed.BindHandler(e, tf, mockApp)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users/feed", nil, nil)
			AssertJSONResponseType[gen.UsersFeedResponse](t, actual)
		})

		t.Run("GET /v1/users/feedにfirstを指定でき、afterを省略した先頭ページが取得できる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersFeed(gomock.Any(), gomock.Any()).
				Return(&user.UserFeedView{Items: []user.UserView{expectedDTO}, NextCursor: nil}, nil)

			feed.BindHandler(e, tf, mockApp)

			// first クエリパラメータがハンドラまで届き、200 が返ることを確認する。
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users/feed?first=10", nil, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/users/feedがErrInvalidArgumentで400を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersFeed(gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrInvalidArgument)

			feed.BindHandler(e, tf, mockApp)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users/feed", nil, nil)
			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})
	})
}
