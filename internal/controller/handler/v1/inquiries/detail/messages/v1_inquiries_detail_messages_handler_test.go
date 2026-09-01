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
	"go-boilerplate/internal/controller/handler/v1/inquiries/detail/messages/gen"
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
	authn, err := auth.New("operator-jane", "issuer", nil, nil)
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
		AuthorKind: "operator",
		Body:       "確認します",
		Sequence:   2,
		CreatedAt:  time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})

	routes := e.Router().Routes()
	testassert.AssertEchoRouterPath(t, "/v1/inquiries/:inquiryId/messages", routes)
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet, http.MethodPost}, routes)
}

func Test_server_GetInquiriesDetailMessages(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パスの問い合わせを指定して履歴を取得する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			inquiryID := uuidtestkit.NewTestFromSalt(t, "inquiry")

			var captured inquiryuc.OperatorHistoryParams
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetInquiryHistory(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, params inquiryuc.OperatorHistoryParams) (*inquiryuc.HistoryView, error) {
					captured = params
					return &inquiryuc.HistoryView{
						InquiryID:    inquiryID,
						Messages:     []inquiryuc.MessageView{newTestMessageView(t)},
						StreamCursor: 2,
					}, nil
				},
			)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			actual, err := s.GetInquiriesDetailMessages(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "admin")),
				gen.GetInquiriesDetailMessagesRequestObject{InquiryId: inquiryID.ToPrimitive()},
			)

			require.NoError(t, err)
			response, ok := actual.(gen.GetInquiriesDetailMessages200JSONResponse)
			require.True(t, ok)
			assert.Len(t, response.Messages, 1)
			assert.Equal(t, inquiryID, captured.InquiryID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証されていなければユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()
			uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			_, err := s.GetInquiriesDetailMessages(
				context.Background(), gen.GetInquiriesDetailMessagesRequestObject{},
			)

			require.Error(t, err)
		})
	})
}

func Test_server_PostInquiriesDetailMessages(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("回答者を認証主体の内部IDにして追加する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			inquiryID := uuidtestkit.NewTestFromSalt(t, "inquiry")
			operatorID := uuidtestkit.NewTestFromSalt(t, "admin")

			var captured inquiryuc.ReplyParams
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().Reply(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, params inquiryuc.ReplyParams) (inquiryuc.MessageView, error) {
					captured = params
					return newTestMessageView(t), nil
				},
			)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			actual, err := s.PostInquiriesDetailMessages(
				authnContext(t, operatorID),
				gen.PostInquiriesDetailMessagesRequestObject{
					InquiryId: inquiryID.ToPrimitive(),
					Body:      &gen.InquiryMessagePostRequest{Body: "確認します"},
				},
			)

			require.NoError(t, err)
			_, ok := actual.(gen.PostInquiriesDetailMessages201JSONResponse)
			assert.True(t, ok)
			assert.Equal(t, inquiryID, captured.InquiryID)
			assert.Equal(t, operatorID, captured.OperatorID)
			assert.Equal(t, "確認します", captured.Body)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユースケースの失敗をそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().Reply(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(inquiryuc.MessageView{}, apperror.ErrNotFound)

			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			_, err := s.PostInquiriesDetailMessages(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "admin")),
				gen.PostInquiriesDetailMessagesRequestObject{
					Body: &gen.InquiryMessagePostRequest{Body: "確認します"},
				},
			)

			require.ErrorIs(t, err, apperror.ErrNotFound)
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

			response := toHistoryResponse(&inquiryuc.HistoryView{
				InquiryID:    inquiryID,
				Messages:     []inquiryuc.MessageView{newTestMessageView(t)},
				StreamCursor: 2,
			})

			require.NotNil(t, response.InquiryId)
			assert.Equal(t, inquiryID.ToPrimitive(), *response.InquiryId)
			assert.Len(t, response.Messages, 1)
			assert.Equal(t, int64(2), response.StreamCursor)
		})

		t.Run("問い合わせが決まらない履歴ではinquiryIdをnullにする", func(t *testing.T) {
			t.Parallel()

			response := toHistoryResponse(&inquiryuc.HistoryView{})

			assert.Nil(t, response.InquiryId)
			assert.NotNil(t, response.Messages)
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

		t.Run("回答の各項目を応答の型へ写す", func(t *testing.T) {
			t.Parallel()
			view := newTestMessageView(t)

			got := toInquiryMessage(view)

			assert.Equal(t, view.ID.ToPrimitive(), got.Id)
			assert.Equal(t, gen.InquiryMessageAuthorKind("operator"), got.AuthorKind)
			assert.Equal(t, int64(2), got.Sequence)
		})
	})
}
