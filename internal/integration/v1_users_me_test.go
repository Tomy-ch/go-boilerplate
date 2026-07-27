package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	detail "go-boilerplate/internal/controller/handler/v1/users/detail"
	detailgen "go-boilerplate/internal/controller/handler/v1/users/detail/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const mePath = "/v1/users/me"

func TestV1UsersMe_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/users/meが認証ユーザー自身のUserResponseを返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			expectedDTO := user.UserView{
				FirstName: "Me", LastName: "Self", Email: "me@example.com", Phone: "09000000000",
				PostalCode: "150-0041", PrefectureName: "Tokyo", City: "Shibuya", Street: "1-2-3",
			}
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedDTO, nil)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-self"))

			actual := StartServer(t, e).DoJSON(http.MethodGet, mePath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[detailgen.UserResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/users/meが認証コンテキスト不在で401を返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			// Authn を注入しない（MakeAvailableUserID を呼ばない）ため、handler が ErrUnauthenticatedUser を返す。
			mockApp := mock_user.NewMockUsecase(ctrl)

			detail.BindHandler(e, tf, mockApp)

			actual := StartServer(t, e).DoJSON(http.MethodGet, mePath, nil, http.Header{})
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("GET /v1/users/meがErrNotFoundで404を返す(退会検知の本筋は#583の認証段)", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(user.UserView{}, apperror.ErrNotFound)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-404"))

			actual := StartServer(t, e).DoJSON(http.MethodGet, mePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})
	})
}
