package feed

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/v1/users/feed/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users/feed"

func newServer(t *testing.T) (*server, *mock_user.MockUsecase) {
	t.Helper()
	mockApp := mock_user.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockApp}, mockApp
}

// wantUserResponse は、本番 toUserResponse とは独立な検証用オラクル（フィールド取り違え検出）。
func wantUserResponse(dto user.UserView) gen.UserResponse {
	return gen.UserResponse{
		FirstName:  dto.FirstName,
		LastName:   dto.LastName,
		Email:      types.Email(dto.Email),
		Phone:      dto.Phone,
		PostalCode: dto.PostalCode,
		Prefecture: dto.PrefectureName,
		City:       dto.City,
		Street:     dto.Street,
		Building:   dto.Building,
		DeletedAt:  dto.DeletedAt,
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockApp := mock_user.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockApp)

	// /v1/users/feed (GET) が登録される。
	routes := e.Router().Routes()

	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, targetPath, routes[0].Path)
}

func Test_server_GetUsersFeed(t *testing.T) {
	t.Parallel()

	expectedDTO1 := user.UserView{
		FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "1234567890",
		PostalCode: "100-0001", PrefectureName: "Tokyo", City: "Chiyoda", Street: "1-1", Building: new("B1"),
	}
	expectedDTO2 := user.UserView{
		FirstName: "User2", LastName: "Two", Email: "user2@example.com", Phone: "0987654321",
		PostalCode: "200-0002", PrefectureName: "Osaka", City: "Kita", Street: "2-2", Building: new("B2"),
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("次ページが無い場合、NextCursorがnilでHasNextがfalseのレスポンスが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			req := gen.GetUsersFeedRequestObject{Params: gen.GetUsersFeedParams{}}

			dtos := []user.UserView{expectedDTO1, expectedDTO2}
			mockApp.EXPECT().
				ListUsersFeed(gomock.Any(), gomock.AssignableToTypeOf(&paging.Cursor{})).
				Return(&user.UserFeedView{Items: dtos, NextCursor: nil}, nil)

			resp, err := s.GetUsersFeed(context.Background(), req)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsersFeed200JSONResponse)
			require.True(t, ok)

			expectedResponse := gen.UsersFeedResponse{
				Users:      []gen.UserResponse{wantUserResponse(expectedDTO1), wantUserResponse(expectedDTO2)},
				NextCursor: nil,
				HasNext:    false,
			}
			assert.Equal(t, expectedResponse, gen.UsersFeedResponse(actual))
		})

		t.Run("次ページがある場合、NextCursorが設定されHasNextがtrueのレスポンスが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			// NewCursor を通過させるため、正しくエンコードされた不透明カーソルを渡す。
			after := paging.EncodeCursor("2025-01-01T00:00:00Z", uuid.NewTestFromSalt(t, "feed_after").String())
			first := 20
			req := gen.GetUsersFeedRequestObject{
				Params: gen.GetUsersFeedParams{After: &after, First: &first},
			}

			nextCursor := "next-opaque-cursor"
			dtos := []user.UserView{expectedDTO1}
			mockApp.EXPECT().
				ListUsersFeed(gomock.Any(), gomock.AssignableToTypeOf(&paging.Cursor{})).
				Return(&user.UserFeedView{Items: dtos, NextCursor: &nextCursor}, nil)

			resp, err := s.GetUsersFeed(context.Background(), req)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsersFeed200JSONResponse)
			require.True(t, ok)

			expectedResponse := gen.UsersFeedResponse{
				Users:      []gen.UserResponse{wantUserResponse(expectedDTO1)},
				NextCursor: &nextCursor,
				HasNext:    true,
			}
			assert.Equal(t, expectedResponse, gen.UsersFeedResponse(actual))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カーソル生成が失敗した場合、ErrInvalidArgumentが返る", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			// base64 として不正な after を渡し NewCursor を失敗させる。
			invalidAfter := "!!!not-base64!!!"
			req := gen.GetUsersFeedRequestObject{
				Params: gen.GetUsersFeedParams{After: &invalidAfter},
			}

			resp, err := s.GetUsersFeed(context.Background(), req)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("Usecaseがエラーを返した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			req := gen.GetUsersFeedRequestObject{Params: gen.GetUsersFeedParams{}}

			expectedErr := apperror.ErrInternal
			mockApp.EXPECT().
				ListUsersFeed(gomock.Any(), gomock.AssignableToTypeOf(&paging.Cursor{})).
				Return(nil, expectedErr)

			resp, err := s.GetUsersFeed(context.Background(), req)
			require.Nil(t, resp)
			require.ErrorIs(t, err, expectedErr)
		})
	})
}

func Test_toUserResponse(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
