package messages

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
	"go-boilerplate/internal/controller/handler/v1/inquiries/me/messages/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/idempotency"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	mock_inquiryuc "go-boilerplate/internal/usecase/inquiry/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

// authnContext は、内部ユーザー ID を解決済みの認証コンテキストを返すテストヘルパーです。
func authnContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	authn, err := auth.New("user-john-doe", "issuer", nil, nil)
	require.NoError(t, err)

	resolved, err := authn.WithUserID(userID)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *resolved))
	return ctx
}

// newTestMessageView は、応答へ写す元となるメッセージを作ります。
func newTestMessageView(t *testing.T) inquiryuc.MessageView {
	t.Helper()
	return inquiryuc.MessageView{
		ID:         uuidtestkit.NewTestFromSalt(t, "message"),
		InquiryID:  uuidtestkit.NewTestFromSalt(t, "inquiry"),
		AuthorKind: "user",
		Body:       "本文",
		Sequence:   3,
		CreatedAt:  time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})

	routes := e.Router().Routes()
	testassert.AssertEchoRouterPath(t, "/v1/inquiries/me/messages", routes)
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet, http.MethodPost}, routes)
}

func Test_server_GetInquiriesMeMessages(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("履歴を取得して応答へ写す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetHistory(gomock.Any(), gomock.Any()).Return(&inquiryuc.HistoryView{
				InquiryID:    uuidtestkit.NewTestFromSalt(t, "inquiry"),
				Messages:     []inquiryuc.MessageView{newTestMessageView(t)},
				StreamCursor: 3,
			}, nil)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}
			ctx := authnContext(t, uuidtestkit.NewTestFromSalt(t, "user"))

			actual, err := s.GetInquiriesMeMessages(ctx, gen.GetInquiriesMeMessagesRequestObject{})

			require.NoError(t, err)
			response, ok := actual.(gen.GetInquiriesMeMessages200JSONResponse)
			require.True(t, ok)
			assert.Len(t, response.Messages, 1)
			assert.Equal(t, int64(3), response.StreamCursor)
		})

		t.Run("開始位置と件数をユースケースへ渡す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			after, first := int64(5), 10

			var captured inquiryuc.HistoryParams
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetHistory(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params inquiryuc.HistoryParams) (*inquiryuc.HistoryView, error) {
					captured = params
					return &inquiryuc.HistoryView{}, nil
				},
			)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			_, err := s.GetInquiriesMeMessages(authnContext(t, userID), gen.GetInquiriesMeMessagesRequestObject{
				Params: gen.GetInquiriesMeMessagesParams{AfterSequence: &after, First: &first},
			})

			require.NoError(t, err)
			assert.Equal(t, userID, captured.UserID)
			require.NotNil(t, captured.AfterSequence)
			assert.Equal(t, int64(5), *captured.AfterSequence)
			require.NotNil(t, captured.First)
			assert.Equal(t, 10, *captured.First)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証されていなければユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()
			uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.GetInquiriesMeMessages(context.Background(), gen.GetInquiriesMeMessagesRequestObject{})

			require.Error(t, err)
		})
	})
}

func Test_server_PostInquiriesMeMessages(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本文と主体を渡して追加した内容を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			var captured inquiryuc.AppendMessageParams
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().AppendMessage(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params inquiryuc.AppendMessageParams) (inquiryuc.MessageView, error) {
					captured = params
					return newTestMessageView(t), nil
				},
			)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			actual, err := s.PostInquiriesMeMessages(authnContext(t, userID), gen.PostInquiriesMeMessagesRequestObject{
				Body: &gen.InquiryMessagePostRequest{Body: "本文"},
			})

			require.NoError(t, err)
			_, ok := actual.(gen.PostInquiriesMeMessages201JSONResponse)
			assert.True(t, ok)
			assert.Equal(t, userID, captured.UserID)
			assert.Equal(t, "user-john-doe", captured.Subject)
			assert.Equal(t, "本文", captured.Body)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユースケースの失敗をそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().AppendMessage(gomock.Any(), gomock.Any()).
				Return(inquiryuc.MessageView{}, apperror.ErrConflict)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			_, err := s.PostInquiriesMeMessages(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "user")),
				gen.PostInquiriesMeMessagesRequestObject{Body: &gen.InquiryMessagePostRequest{Body: "本文"}},
			)

			require.ErrorIs(t, err, apperror.ErrConflict)
		})
	})
}

func Test_toHistoryResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メッセージと現在位置を応答の型へ写す", func(t *testing.T) {
			t.Parallel()
			inquiryID := uuidtestkit.NewTestFromSalt(t, "inquiry")
			next := int64(9)

			response := toHistoryResponse(&inquiryuc.HistoryView{
				InquiryID:         inquiryID,
				Messages:          []inquiryuc.MessageView{newTestMessageView(t)},
				NextAfterSequence: &next,
				StreamCursor:      12,
			})

			require.NotNil(t, response.InquiryId)
			assert.Equal(t, inquiryID.ToPrimitive(), *response.InquiryId)
			assert.Len(t, response.Messages, 1)
			require.NotNil(t, response.NextAfterSequence)
			assert.Equal(t, int64(9), *response.NextAfterSequence)
			assert.Equal(t, int64(12), response.StreamCursor)
		})

		t.Run("問い合わせを持たない履歴ではinquiryIdをnullにする", func(t *testing.T) {
			t.Parallel()

			response := toHistoryResponse(&inquiryuc.HistoryView{})

			assert.Nil(t, response.InquiryId)
			assert.NotNil(t, response.Messages)
			assert.Empty(t, response.Messages)
		})
	})
}

func Test_toMessageResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メッセージを応答の封筒へ入れる", func(t *testing.T) {
			t.Parallel()
			view := newTestMessageView(t)

			response := toMessageResponse(view)

			assert.Equal(t, view.ID.ToPrimitive(), response.Message.Id)
		})
	})
}

func Test_toInquiryMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メッセージの各項目を応答の型へ写す", func(t *testing.T) {
			t.Parallel()
			view := newTestMessageView(t)

			got := toInquiryMessage(view)

			assert.Equal(t, view.ID.ToPrimitive(), got.Id)
			assert.Equal(t, view.InquiryID.ToPrimitive(), got.InquiryId)
			assert.Equal(t, gen.InquiryMessageAuthorKind("user"), got.AuthorKind)
			assert.Equal(t, "本文", got.Body)
			assert.Equal(t, int64(3), got.Sequence)
			assert.Equal(t, view.CreatedAt, got.CreatedAt)
		})
	})
}
