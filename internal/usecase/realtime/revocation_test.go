package realtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
)

func newRevoker(t *testing.T) (*accessRevoker, *mock_realtime.MockStreamTicketStore, *mock_realtime.MockRevocationNotifier) {
	t.Helper()

	ctrl := gomock.NewController(t)
	tickets := mock_realtime.NewMockStreamTicketStore(ctrl)
	notifier := mock_realtime.NewMockRevocationNotifier(ctrl)

	return &accessRevoker{tickets: tickets, notifier: notifier, tracer: observability.NewNoopTracerFactory(t).Usecase()}, tickets, notifier
}

func TestNewAccessRevoker(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	r := NewAccessRevoker(mock_realtime.NewMockStreamTicketStore(ctrl), mock_realtime.NewMockRevocationNotifier(ctrl), observability.NewNoopTracerFactory(t))
	assert.NotNil(t, r)
}

func Test_accessRevoker_Revoke(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ticketを無効にしてから全instanceへ通知する", func(t *testing.T) {
			t.Parallel()

			r, tickets, notifier := newRevoker(t)
			gomock.InOrder(
				tickets.EXPECT().Invalidate(gomock.Any(), "subject-1", rt.StreamID("stream-1")).Return(nil),
				notifier.EXPECT().NotifyRevoked(gomock.Any(), "subject-1", rt.StreamID("stream-1")).Return(nil),
			)

			require.NoError(t, r.Revoke(t.Context(), "subject-1", "stream-1"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無効化に失敗すれば通知せずstoreのエラーを返す", func(t *testing.T) {
			t.Parallel()

			r, tickets, _ := newRevoker(t)
			tickets.EXPECT().Invalidate(gomock.Any(), "subject-1", rt.StreamID("stream-1")).Return(errStoreOff)

			require.ErrorIs(t, r.Revoke(t.Context(), "subject-1", "stream-1"), apperror.ErrUnavailable)
		})

		t.Run("通知に失敗しても無効化は済んでおりそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			r, tickets, notifier := newRevoker(t)
			tickets.EXPECT().Invalidate(gomock.Any(), "subject-1", rt.StreamID("stream-1")).Return(nil)
			notifier.EXPECT().NotifyRevoked(gomock.Any(), "subject-1", rt.StreamID("stream-1")).Return(errStoreOff)

			require.ErrorIs(t, r.Revoke(t.Context(), "subject-1", "stream-1"), apperror.ErrUnavailable)
		})
	})
}
