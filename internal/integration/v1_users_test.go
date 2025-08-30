package integration

import (
	"net/http"
	"testing"

	v1users "boilerplate-go/internal/controller/handler/v1/users"
	"boilerplate-go/internal/controller/handler/v1/users/gen"
	useruc "boilerplate-go/internal/usecase/user"
	mock_useruc "boilerplate-go/internal/usecase/user/mock"

	"github.com/labstack/echo/v4"
	"go.uber.org/mock/gomock"
)

func TestV1Users_Integration(t *testing.T) {
	t.Parallel()

	t.Run("GET /v1/usersのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e := echo.New()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockApp := mock_useruc.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			GetAllUsers(gomock.Any(), gomock.Any()).
			Return([]useruc.DTO{}, nil)

		v1users.BindHandler(e, mockApp)

		actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users", nil, nil)
		AssertJSONResponse(t, gen.ResponseV1Users{}, actual)
	})
}
