package merge

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/carts/merge/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	cartuc "go-boilerplate/internal/usecase/cart"
	mock_cartuc "go-boilerplate/internal/usecase/cart/mock"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// testSessionToken は、ヘッダで受け取る引き継ぎ元のトークンのサンプル値。
const testSessionToken = "abcdefghijklmnopqrstuvwxyz0123456789-_ABCDE"

// authnContext は、内部ユーザー ID を解決済みの認証コンテキストを返すテストヘルパーです。
func authnContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	authn, err := auth.New("subject", "issuer", nil, nil)
	require.NoError(t, err)

	resolved, err := authn.WithUserID(userID)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *resolved))
	return ctx
}

// newMergeRequest は、引き継ぎ元のトークンを載せたリクエストを組み立てるテストヘルパーです。
func newMergeRequest() gen.PostCartsMeMergeRequestObject {
	return gen.PostCartsMeMergeRequestObject{
		Params: gen.PostCartsMeMergeParams{XCartSession: testSessionToken},
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	uc := mock_cartuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, uc, idempotency.Deps{})

	routes := e.Router().Routes()
	testassert.AssertEchoRouterPath(t, "/v1/carts/me/merge", routes)
	testassert.AssertEchoRouterMethods(t, []string{http.MethodPost}, routes)
}

func Test_server_PostCartsMeMerge(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体とヘッダのトークンをユースケースへ渡し200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuidtestkit.NewTestFromSalt(t, "merge_h_user")
			clamped := uuidtestkit.NewTestFromSalt(t, "merge_h_clamped")
			uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, params cartuc.MergeOnLoginParams) (cartuc.MergeCartResult, error) {
					assert.Equal(t, userID, params.UserID)
					assert.Equal(t, testSessionToken, params.SessionToken)
					return cartuc.MergeCartResult{Clamped: []uuid.UUID{clamped}}, nil
				})

			resp, err := s.PostCartsMeMerge(authnContext(t, userID), newMergeRequest())
			require.NoError(t, err)

			r, ok := resp.(gen.PostCartsMeMerge200JSONResponse)
			require.True(t, ok)
			require.Len(t, r.Clamped, 1)
			assert.Equal(t, clamped.ToPrimitive(), r.Clamped[0])
			assert.Empty(t, r.Dropped)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証コンテキストが無い場合はユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).Times(0)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.PostCartsMeMerge(context.Background(), newMergeRequest())

			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})

		t.Run("認証済みだが内部UserIDが未解決の場合はユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).Times(0)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			_, err = s.PostCartsMeMerge(ctx, newMergeRequest())

			require.ErrorIs(t, err, auth.ErrUserIDUnresolved)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).
				Return(cartuc.MergeCartResult{}, apperror.ErrInternal)

			userID := uuidtestkit.NewTestFromSalt(t, "merge_h_err")
			_, err := s.PostCartsMeMerge(authnContext(t, userID), newMergeRequest())

			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_toCartMergeResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失われた分をレスポンスへ写す", func(t *testing.T) {
			t.Parallel()

			clamped := uuidtestkit.NewTestFromSalt(t, "resp_clamped")
			dropped := uuidtestkit.NewTestFromSalt(t, "resp_dropped")

			actual := toCartMergeResponse(cartuc.MergeCartResult{
				Clamped: []uuid.UUID{clamped},
				Dropped: []uuid.UUID{dropped},
			})

			require.Len(t, actual.Clamped, 1)
			require.Len(t, actual.Dropped, 1)
			assert.Equal(t, clamped.ToPrimitive(), actual.Clamped[0])
			assert.Equal(t, dropped.ToPrimitive(), actual.Dropped[0])
		})

		t.Run("失われた分が無ければnullではなく空配列になる", func(t *testing.T) {
			t.Parallel()

			actual := toCartMergeResponse(cartuc.MergeCartResult{})

			assert.NotNil(t, actual.Clamped)
			assert.NotNil(t, actual.Dropped)
			assert.Empty(t, actual.Clamped)
			assert.Empty(t, actual.Dropped)
		})
	})
}

func Test_toUUIDs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("応答の型へ変換する", func(t *testing.T) {
			t.Parallel()

			id := uuidtestkit.NewTestFromSalt(t, "to_uuids")

			actual := toUUIDs([]uuid.UUID{id})

			require.Len(t, actual, 1)
			assert.Equal(t, id.ToPrimitive(), actual[0])
		})

		t.Run("空でもnullにならない", func(t *testing.T) {
			t.Parallel()

			actual := toUUIDs(nil)

			assert.NotNil(t, actual)
			assert.Empty(t, actual)
		})
	})
}
