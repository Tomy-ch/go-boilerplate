package withdrawalarchive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	workerbd "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/xerrors"
)

// testWithdrawnUserID は、退会イベント payload に載るユーザー ID です。
const testWithdrawnUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

// newHandlerUnderTest は、mock を差した Handler を生成します。
func newHandlerUnderTest(t *testing.T) (*handler, *mock_user.MockArchiveUsecase) {
	t.Helper()

	archive := mock_user.NewMockArchiveUsecase(gomock.NewController(t))
	return newHandler(archive, observability.NewNoopTracerFactory(t), logging.NewTestLogger(t)), archive
}

// withdrawnMessage は、退会イベントとして届くメッセージを組み立てます。
func withdrawnMessage(body string) workerbd.Message {
	return workerbd.Message{
		ID:         "broker-message-id",
		Body:       []byte(body),
		Attributes: map[string]string{workerbd.AttrEventType: "user.withdrawn.v1"},
	}
}

func Test_newHandler(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユースケースと計装から Handler を生成する", func(t *testing.T) {
			t.Parallel()

			got, archive := newHandlerUnderTest(t)

			require.NotNil(t, got)
			assert.Same(t, archive, got.archive)
		})
	})
}

func Test_handler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("退会イベントの payload をそのまま保存へ渡す", func(t *testing.T) {
			t.Parallel()

			h, archive := newHandlerUnderTest(t)
			const body = `{"userId":"` + testWithdrawnUserID + `","deletedAt":"2026-07-29T12:00:00Z"}`
			archive.EXPECT().
				ArchiveWithdrawal(gomock.Any(), user.ArchiveWithdrawalParams{
					UserID:  testWithdrawnUserID,
					Payload: []byte(body),
				}).
				Return("withdrawals/"+testWithdrawnUserID+".json", nil)

			err := h.Handle(t.Context(), withdrawnMessage(body))

			require.NoError(t, err)
		})

		t.Run("種別が異なるメッセージは保存せず成功を返す", func(t *testing.T) {
			t.Parallel()
			// 1 つのキューに全種別が流れるため、他種別を Permanent 扱いにすると DLQ が埋まる。
			h, _ := newHandlerUnderTest(t)
			m := withdrawnMessage(`{"purchaseId":"p1"}`)
			m.Attributes[workerbd.AttrEventType] = "purchase.created.v1"

			err := h.Handle(t.Context(), m)

			require.NoError(t, err)
		})

		t.Run("種別属性が無いメッセージは保存せず成功を返す", func(t *testing.T) {
			t.Parallel()
			// 属性を載せる前に publish された残留メッセージで DLQ を埋めないための扱い。
			h, _ := newHandlerUnderTest(t)
			m := withdrawnMessage(`{"userId":"` + testWithdrawnUserID + `"}`)
			m.Attributes = nil

			err := h.Handle(t.Context(), m)

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("復元できない payload を Permanent として分類する", func(t *testing.T) {
			t.Parallel()

			h, _ := newHandlerUnderTest(t)

			err := h.Handle(t.Context(), withdrawnMessage("not-json"))

			require.ErrorIs(t, err, apperror.ErrPermanent)
		})

		t.Run("入力不正による保存失敗を Permanent として分類する", func(t *testing.T) {
			t.Parallel()
			// 何度配送されても同じ結果になるため、再配送させず退避側へ回す。
			h, archive := newHandlerUnderTest(t)
			archive.EXPECT().
				ArchiveWithdrawal(gomock.Any(), gomock.Any()).
				Return("", xerrors.Wrap(apperror.ErrValidation, "user id must be a uuid"))

			err := h.Handle(t.Context(), withdrawnMessage(`{"userId":"not-a-uuid"}`))

			require.ErrorIs(t, err, apperror.ErrPermanent)
		})

		t.Run("保存の一時失敗は分類せずそのまま返す", func(t *testing.T) {
			t.Parallel()
			// engine 既定の Retryable として再配送させる。
			h, archive := newHandlerUnderTest(t)
			archive.EXPECT().
				ArchiveWithdrawal(gomock.Any(), gomock.Any()).
				Return("", xerrors.Wrap(apperror.ErrUnavailable, "storage down"))

			err := h.Handle(t.Context(), withdrawnMessage(`{"userId":"`+testWithdrawnUserID+`"}`))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.NotErrorIs(t, err, apperror.ErrPermanent)
		})
	})
}
