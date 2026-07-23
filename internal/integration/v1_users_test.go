package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	v1users "go-boilerplate/internal/controller/handler/v1/users"
	"go-boilerplate/internal/controller/handler/v1/users/gen"
	"go-boilerplate/internal/observability"
	clocktest "go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1Users_Integration(t *testing.T) {
	t.Parallel()

	expectedDTO := user.UserView{FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "1234567890"}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/usersがユーザーリストを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersWithTotal(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&user.UserListView{Items: []user.UserView{expectedDTO}, Total: 1}, nil)

			v1users.BindHandler(e, tf, mockApp, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users", nil, nil)
			AssertJSONResponseType[gen.UsersResponse](t, actual)
		})

		t.Run("POST /v1/usersがユーザー作成を行い201を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				CreateUser(gomock.Any(), gomock.Any()).
				Return(user.UserView{Email: "new@example.com"}, nil)

			v1users.BindHandler(e, tf, mockApp, idempotency.Deps{})

			req := gen.PostUsersRequestObject{
				Body: &gen.PostUsersJSONRequestBody{
					FirstName:  "First",
					LastName:   "Last",
					Email:      types.Email("new@example.com"),
					Phone:      "09000000000",
					PostalCode: "123-4567",
					Prefecture: "Tokyo",
					City:       "Shibuya",
					Street:     "1-1-1",
					Building:   new("Building"),
				},
			}

			uuid, err := uuid.Parse("d1f64798-7321-242b-e4ff-115f6a0b7803")
			require.NoError(t, err)
			headers := MakeAvailableUserID(t, e, uuid)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/users", req.Body, headers)
			assert.Equal(t, http.StatusCreated, actual.StatusCode)
		})

		t.Run("POST /v1/usersがIdempotency-Key付きでclaim→completeし201を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				CreateUser(gomock.Any(), gomock.Any()).
				Return(user.UserView{Email: "idem@example.com"}, nil)

			// 実 Deps を組んだ BindHandler に Idempotency-Key 付きリクエストを通し、
			// middleware→Run 配線（claim→complete）が実スタックで動くことを検証する。
			store := mock_idempotency.NewMockStore(ctrl)
			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)
			store.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(nil)
			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			clk := clocktest.NewMockClock(t, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))

			v1users.BindHandler(e, tf, mockApp, idempotency.Deps{Txm: txm, Store: store, Clock: clk})

			req := gen.PostUsersRequestObject{
				Body: &gen.PostUsersJSONRequestBody{
					FirstName:  "First",
					LastName:   "Last",
					Email:      types.Email("idem@example.com"),
					Phone:      "09000000000",
					PostalCode: "123-4567",
					Prefecture: "Tokyo",
					City:       "Shibuya",
					Street:     "1-1-1",
					Building:   new("Building"),
				},
			}

			uuid, err := uuid.Parse("e2f64798-7321-242b-e4ff-115f6a0b7804")
			require.NoError(t, err)
			headers := MakeAvailableUserID(t, e, uuid)
			headers.Set("Idempotency-Key", "integration-key-1")
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/users", req.Body, headers)
			assert.Equal(t, http.StatusCreated, actual.StatusCode)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/usersがErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersWithTotal(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrInternal)

			v1users.BindHandler(e, tf, mockApp, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
