package inquiry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"
)

// newTestAuthn は、テスト用の認証結果を組み立てます。
func newTestAuthn(t *testing.T) *authbd.Authn {
	t.Helper()
	authn, err := authbd.New("user-john-doe", "http://localhost:2010/default", nil, nil)
	require.NoError(t, err)
	return authn
}

func Test_usecase_IssueStreamTicket(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("自分の問い合わせを宛先にしたticketを発行する", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			i := newTestInquiry(t, userID)

			var captured ucrealtime.IssueTicketInput
			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(i, nil)
			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(4), true, nil)
			d.tickets.EXPECT().Issue(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ucrealtime.IssueTicketInput) (ucrealtime.TicketView, error) {
					captured = in
					return ucrealtime.TicketView{Value: "raw-ticket", ExpiresAt: baseTime}, nil
				},
			)

			view, err := u.IssueStreamTicket(context.Background(), IssueStreamTicketParams{
				UserID: userID, Subject: "user-john-doe",
			})

			require.NoError(t, err)
			assert.Equal(t, "raw-ticket", view.Ticket)
			assert.Equal(t, i.ID().String(), view.StreamID)
			assert.Equal(t, rt.StreamID(i.ID().String()), captured.Destination)
			assert.Equal(t, conversationScope, captured.Scope)
			assert.Equal(t, rt.Sequence(4), captured.InitialCursor)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購読する問い合わせが無ければNotFoundを返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)

			_, err := u.IssueStreamTicket(context.Background(), IssueStreamTicketParams{UserID: userID})

			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

func Test_usecase_IssueFeedTicket(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("feedを宛先にしたticketを発行する", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			var captured ucrealtime.IssueTicketInput
			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.sequences.EXPECT().Current(gomock.Any(), feedStreamID).Return(rt.Sequence(0), false, nil)
			d.tickets.EXPECT().Issue(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ucrealtime.IssueTicketInput) (ucrealtime.TicketView, error) {
					captured = in
					return ucrealtime.TicketView{Value: "raw-feed-ticket", ExpiresAt: baseTime}, nil
				},
			)

			view, err := u.IssueFeedTicket(context.Background(), newTestAuthn(t))

			require.NoError(t, err)
			assert.Equal(t, feedStreamID.String(), view.StreamID)
			assert.Equal(t, feedScope, captured.Scope)
			assert.Equal(t, rt.Sequence(0), captured.InitialCursor)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者でなければ発行しない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(apperror.ErrPermissionDenied)

			_, err := u.IssueFeedTicket(context.Background(), newTestAuthn(t))

			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})
	})
}

func Test_usecase_issueTicket(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未採番のstreamでは開始位置を0にする", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			var captured ucrealtime.IssueTicketInput
			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(7), false, nil)
			d.tickets.EXPECT().Issue(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ucrealtime.IssueTicketInput) (ucrealtime.TicketView, error) {
					captured = in
					return ucrealtime.TicketView{Value: "raw", ExpiresAt: baseTime}, nil
				},
			)

			_, err := u.issueTicket(context.Background(), "user-john-doe", rt.StreamID("s"), conversationScope)

			require.NoError(t, err)
			assert.Equal(t, rt.Sequence(0), captured.InitialCursor)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現在位置を読めなければ発行しない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			wantErr := xerrors.New("current failed")

			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(0), false, wantErr)

			_, err := u.issueTicket(context.Background(), "user-john-doe", rt.StreamID("s"), conversationScope)

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("発行の失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			wantErr := xerrors.New("issue failed")

			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), true, nil)
			d.tickets.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(ucrealtime.TicketView{}, wantErr)

			_, err := u.issueTicket(context.Background(), "user-john-doe", rt.StreamID("s"), conversationScope)

			require.ErrorIs(t, err, wantErr)
		})
	})
}
