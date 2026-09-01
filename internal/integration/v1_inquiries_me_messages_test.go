package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	messages "go-boilerplate/internal/controller/handler/v1/inquiries/me/messages"
	"go-boilerplate/internal/controller/handler/v1/inquiries/me/messages/gen"
	"go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/idempotency"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	mock_inquiryuc "go-boilerplate/internal/usecase/inquiry/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

const meInquiryMessagesPath = "/v1/inquiries/me/messages"

func TestV1InquiriesMeMessages_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("履歴の取得はstreamCursorを含めて200で返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			inquiryID := uuidtestkit.NewTestFromSalt(t, "int_inq")

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetHistory(gomock.Any(), gomock.Any()).Return(&inquiryuc.HistoryView{
				InquiryID: inquiryID,
				Messages: []inquiryuc.MessageView{{
					ID:         uuidtestkit.NewTestFromSalt(t, "int_msg"),
					InquiryID:  inquiryID,
					AuthorKind: "user",
					Body:       "届きません",
					Sequence:   1,
					CreatedAt:  time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
				}},
				StreamCursor: 1,
			}, nil)

			messages.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_inq_user"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, meInquiryMessagesPath, nil, headers)

			require.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.InquiryHistoryResponse](t, actual)
		})

		t.Run("投稿は本文を渡すと201で追加したメッセージを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			inquiryID := uuidtestkit.NewTestFromSalt(t, "int_inq")

			var captured inquiryuc.AppendMessageParams
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().AppendMessage(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params inquiryuc.AppendMessageParams) (inquiryuc.MessageView, error) {
					captured = params
					return inquiryuc.MessageView{
						ID:         uuidtestkit.NewTestFromSalt(t, "int_msg"),
						InquiryID:  inquiryID,
						AuthorKind: "user",
						Body:       params.Body,
						Sequence:   1,
						CreatedAt:  time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
					}, nil
				},
			)

			messages.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})

			userID := uuidtestkit.NewTestFromSalt(t, "int_inq_user")
			headers := MakeAvailableUserID(t, e, userID)
			actual := StartServer(t, e).DoJSON(
				http.MethodPost, meInquiryMessagesPath, map[string]string{"body": "届きません"}, headers,
			)

			require.Equal(t, http.StatusCreated, actual.StatusCode)

			var body gen.InquiryMessageResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			assert.Equal(t, "user", string(body.Message.AuthorKind))
			assert.Equal(t, "届きません", captured.Body)
			assert.Equal(t, userID, captured.UserID)
			assert.NotEmpty(t, captured.Subject)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証の履歴取得は401を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			uc := mock_inquiryuc.NewMockUsecase(gomock.NewController(t))
			messages.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})
			UseAppErrorHandler(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, meInquiryMessagesPath, nil, nil)

			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("空の本文で投稿すると422を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().AppendMessage(gomock.Any(), gomock.Any()).
				Return(inquiryuc.MessageView{}, inquirymessage.ErrEmptyBody)

			messages.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})
			UseAppErrorHandler(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_inq_user"))
			actual := StartServer(t, e).DoJSON(
				http.MethodPost, meInquiryMessagesPath, map[string]string{"body": ""}, headers,
			)

			AssertErrorResponse(t, actual, http.StatusUnprocessableEntity)
		})
	})
}
