package inquiry

import (
	"context"
	"testing"

	domaininquiry "go-boilerplate/internal/domain/inquiry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"
)

func Test_usecase_GetHistory(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現在位置までのメッセージを返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			i := newTestInquiry(t, userID)

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(i, nil)
			d.sequences.EXPECT().Current(gomock.Any(), rt.StreamID(i.ID().String())).
				Return(rt.Sequence(2), true, nil)
			d.repo.EXPECT().ListMessages(gomock.Any(), i.ID(), gomock.Any()).
				Return([]*domaininquiry.Message{newTestMessage(t, domaininquiry.AuthorKindUser, 1)}, nil)

			view, err := u.GetHistory(context.Background(), HistoryParams{UserID: userID})

			require.NoError(t, err)
			assert.Equal(t, i.ID(), view.InquiryID)
			assert.Len(t, view.Messages, 1)
			assert.Equal(t, int64(2), view.StreamCursor)
			assert.Nil(t, view.NextAfterSequence)
		})

		t.Run("問い合わせを持たない利用者には空の履歴を返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)

			view, err := u.GetHistory(context.Background(), HistoryParams{UserID: userID})

			require.NoError(t, err)
			assert.Empty(t, view.Messages)
			assert.Equal(t, int64(0), view.StreamCursor)
			assert.True(t, view.InquiryID.IsNil())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不在以外の読み出し失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			wantErr := xerrors.New("find failed")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, wantErr)

			_, err := u.GetHistory(context.Background(), HistoryParams{UserID: userID})

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func Test_usecase_historyOf(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現在位置を上限にして読み1件多ければ次ページの位置を返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))

			var captured domaininquiry.HistoryParams
			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(9), true, nil)
			d.repo.EXPECT().ListMessages(gomock.Any(), i.ID(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, params domaininquiry.HistoryParams) ([]*domaininquiry.Message, error) {
					captured = params
					return []*domaininquiry.Message{
						newTestMessage(t, domaininquiry.AuthorKindUser, 1),
						newTestMessage(t, domaininquiry.AuthorKindUser, 2),
					}, nil
				},
			)

			view, err := u.historyOf(context.Background(), i, nil, ptr.To(1))

			require.NoError(t, err)
			assert.Equal(t, int64(9), captured.UpToSequence)
			assert.Equal(t, 2, captured.Limit)
			assert.Len(t, view.Messages, 1)
			require.NotNil(t, view.NextAfterSequence)
			assert.Equal(t, int64(1), *view.NextAfterSequence)
		})

		t.Run("まだ採番されていなければ空の履歴を返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))

			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(0), false, nil)

			view, err := u.historyOf(context.Background(), i, nil, nil)

			require.NoError(t, err)
			assert.Empty(t, view.Messages)
			assert.Equal(t, i.ID(), view.InquiryID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現在位置を読めなければメッセージを読まない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			wantErr := xerrors.New("current failed")

			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(0), false, wantErr)

			_, err := u.historyOf(context.Background(), i, nil, nil)

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("メッセージの読み出し失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			wantErr := xerrors.New("list failed")

			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), true, nil)
			d.repo.EXPECT().ListMessages(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, wantErr)

			_, err := u.historyOf(context.Background(), i, nil, nil)

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func Test_pageOf(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		inquiryID := uuidtestkit.NewTestFromSalt(t, "inquiry")

		t.Run("上限以下なら全件を返し次ページを示さない", func(t *testing.T) {
			t.Parallel()
			messages := []*domaininquiry.Message{newTestMessage(t, domaininquiry.AuthorKindUser, 1)}

			views, next := pageOf(inquiryID, messages, 2)

			assert.Len(t, views, 1)
			assert.Nil(t, next)
		})

		t.Run("上限を超えたら切り詰め末尾の位置を次ページとして返す", func(t *testing.T) {
			t.Parallel()
			messages := []*domaininquiry.Message{
				newTestMessage(t, domaininquiry.AuthorKindUser, 1),
				newTestMessage(t, domaininquiry.AuthorKindUser, 2),
			}

			views, next := pageOf(inquiryID, messages, 1)

			assert.Len(t, views, 1)
			require.NotNil(t, next)
			assert.Equal(t, int64(1), *next)
		})

		t.Run("空なら空の配列を返す", func(t *testing.T) {
			t.Parallel()

			views, next := pageOf(inquiryID, nil, 10)

			assert.Empty(t, views)
			assert.NotNil(t, views)
			assert.Nil(t, next)
		})
	})
}
