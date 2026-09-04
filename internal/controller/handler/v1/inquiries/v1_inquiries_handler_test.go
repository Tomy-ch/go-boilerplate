package inquiries

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/inquiries/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	mock_inquiryuc "go-boilerplate/internal/usecase/inquiry/mock"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

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

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, observability.NewNoopTracerFactory(t), uc)

	routes := e.Router().Routes()
	testassert.AssertEchoRouterPath(t, "/v1/inquiries", routes)
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, routes)
}

func Test_server_GetInquiries(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧を取得して応答へ写す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().ListInquiries(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&inquiryuc.InquiryListView{Items: []inquiryuc.InquirySummaryView{{
					ID:        uuidtestkit.NewTestFromSalt(t, "inquiry"),
					UserID:    uuidtestkit.NewTestFromSalt(t, "user"),
					CreatedAt: time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
				}}}, nil,
			)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}
			ctx := authnContext(t, uuidtestkit.NewTestFromSalt(t, "admin"))

			actual, err := s.GetInquiries(ctx, gen.GetInquiriesRequestObject{})

			require.NoError(t, err)
			response, ok := actual.(gen.GetInquiries200JSONResponse)
			require.True(t, ok)
			assert.Len(t, response.Items, 1)
		})

		t.Run("カーソルと件数をユースケースへ渡す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			after := paging.EncodeCursor(
				time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
				uuidtestkit.NewTestFromSalt(t, "boundary").String(),
			)
			first := 3

			var captured inquiryuc.ListInquiriesParams
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().ListInquiries(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, params inquiryuc.ListInquiriesParams) (*inquiryuc.InquiryListView, error) {
					captured = params
					return &inquiryuc.InquiryListView{}, nil
				},
			)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}
			ctx := authnContext(t, uuidtestkit.NewTestFromSalt(t, "admin"))

			_, err := s.GetInquiries(ctx, gen.GetInquiriesRequestObject{
				Params: gen.GetInquiriesParams{After: &after, First: &first},
			})

			require.NoError(t, err)
			require.NotNil(t, captured.Cursor)
			assert.Equal(t, 3, captured.Cursor.Limit())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証されていなければユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()
			uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.GetInquiries(context.Background(), gen.GetInquiriesRequestObject{})

			require.Error(t, err)
		})

		t.Run("ユースケースの失敗をそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().ListInquiries(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrPermissionDenied)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}
			ctx := authnContext(t, uuidtestkit.NewTestFromSalt(t, "user"))

			_, err := s.GetInquiries(ctx, gen.GetInquiriesRequestObject{})

			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})
	})
}

func Test_toInquiryListResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("要約と次カーソルを応答の型へ写す", func(t *testing.T) {
			t.Parallel()
			id := uuidtestkit.NewTestFromSalt(t, "inquiry")
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			next := "next-cursor"

			response := toInquiryListResponse(&inquiryuc.InquiryListView{
				Items: []inquiryuc.InquirySummaryView{{
					ID:        id,
					UserID:    userID,
					CreatedAt: time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
				}},
				NextCursor: &next,
			})

			require.Len(t, response.Items, 1)
			assert.Equal(t, id.ToPrimitive(), response.Items[0].Id)
			assert.Equal(t, userID.ToPrimitive(), response.Items[0].UserId)
			require.NotNil(t, response.NextCursor)
			assert.Equal(t, next, *response.NextCursor)
		})

		t.Run("空の一覧でもnilではなく空配列を返す", func(t *testing.T) {
			t.Parallel()

			response := toInquiryListResponse(&inquiryuc.InquiryListView{})

			assert.NotNil(t, response.Items)
			assert.Empty(t, response.Items)
			assert.Nil(t, response.NextCursor)
		})
	})
}
