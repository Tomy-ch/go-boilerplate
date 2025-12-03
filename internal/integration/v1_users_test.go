package integration

import (
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/handlertest/testinstance"
	v1users "boilerplate-go/internal/controller/handler/v1/users"
	"boilerplate-go/internal/controller/handler/v1/users/gen"
	"boilerplate-go/internal/usecase/user"
	mock_user "boilerplate-go/internal/usecase/user/mock"

	"go.uber.org/mock/gomock"
)

func TestV1Users_Integration(t *testing.T) {
	t.Parallel()

	t.Run("GET /v1/usersのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e, ctrl, tf, _ := testinstance.NewTestInstanceForBindHandler(t)
		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			GetAllUsers(gomock.Any(), gomock.Any()).
			Return([]user.DTO{}, nil)

		v1users.BindHandler(e, tf, mockApp)

		actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users", nil, nil)
		AssertJSONResponse(t, gen.ResponseV1Users{}, actual)
	})
}
