package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	detailmessages "go-boilerplate/internal/controller/handler/v1/inquiries/detail/messages"
	"go-boilerplate/internal/controller/handler/v1/inquiries/detail/messages/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/idempotency"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	mock_inquiryuc "go-boilerplate/internal/usecase/inquiry/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

// inquiryDetailMessagesPath は、問い合わせ ID を埋めた回答・履歴のパスを返します。
func inquiryDetailMessagesPath(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("/v1/inquiries/%s/messages", uuidtestkit.NewTestFromSalt(t, "int_inq"))
}

func TestV1InquiriesDetailMessages_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("履歴の取得は200でstreamCursorを含めて返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			inquiryID := uuidtestkit.NewTestFromSalt(t, "int_inq")

			var captured inquiryuc.OperatorHistoryParams
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetInquiryHistory(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ any, params inquiryuc.OperatorHistoryParams) (*inquiryuc.HistoryView, error) {
					captured = params
					return &inquiryuc.HistoryView{
						InquiryID:    inquiryID,
						Messages:     []inquiryuc.MessageView{},
						StreamCursor: 0,
					}, nil
				},
			)

			detailmessages.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_admin"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, inquiryDetailMessagesPath(t), nil, headers)

			require.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.InquiryHistoryResponse](t, actual)
			assert.Equal(t, inquiryID, captured.InquiryID)
		})

		t.Run("回答は201で追加したメッセージを返し送り手がoperatorになる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			inquiryID := uuidtestkit.NewTestFromSalt(t, "int_inq")
			operatorID := uuidtestkit.NewTestFromSalt(t, "int_admin")

			var captured inquiryuc.ReplyParams
			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().Reply(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ any, params inquiryuc.ReplyParams) (inquiryuc.MessageView, error) {
					captured = params
					return inquiryuc.MessageView{
						ID:         uuidtestkit.NewTestFromSalt(t, "int_msg"),
						InquiryID:  params.InquiryID,
						AuthorKind: "operator",
						Body:       params.Body,
						Sequence:   2,
						CreatedAt:  time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
					}, nil
				},
			)

			detailmessages.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})

			headers := MakeAvailableUserID(t, e, operatorID)
			actual := StartServer(t, e).DoJSON(
				http.MethodPost, inquiryDetailMessagesPath(t), map[string]string{"body": "確認します"}, headers,
			)

			require.Equal(t, http.StatusCreated, actual.StatusCode)
			AssertJSONResponseType[gen.InquiryMessageResponse](t, actual)
			assert.Equal(t, inquiryID, captured.InquiryID)
			assert.Equal(t, operatorID, captured.OperatorID)
			assert.Equal(t, "確認します", captured.Body)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しない問い合わせへの回答は404を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().Reply(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(inquiryuc.MessageView{}, apperror.ErrNotFound)

			detailmessages.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})
			UseAppErrorHandler(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_admin"))
			actual := StartServer(t, e).DoJSON(
				http.MethodPost, inquiryDetailMessagesPath(t), map[string]string{"body": "確認します"}, headers,
			)

			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("管理者以外の履歴取得は403を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_inquiryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetInquiryHistory(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrPermissionDenied)

			detailmessages.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})
			UseAppErrorHandler(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_user"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, inquiryDetailMessagesPath(t), nil, headers)

			AssertErrorResponse(t, actual, http.StatusForbidden)
		})
	})
}
