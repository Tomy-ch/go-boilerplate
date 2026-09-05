package inquiry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	domaininquiry "go-boilerplate/internal/domain/inquiry"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"
)

// expectAppendSucceeds は、問い合わせが決まった後の追加経路が成功する期待を並べます。
func expectAppendSucceeds(t *testing.T, d deps) {
	t.Helper()
	d.sequences.EXPECT().Allocate(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), nil).Times(2)
	d.repo.EXPECT().CreateMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	d.repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	d.repo.EXPECT().ListMessages(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*domaininquiry.Message{newTestMessage(t, domaininquiry.AuthorKindUser, 1)}, nil)
	d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil).Times(2)
}

func Test_usecase_AppendMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既存の問い合わせへ追加する", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			i := newTestInquiry(t, userID)

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(i, nil)
			expectAppendSucceeds(t, d)

			view, err := u.AppendMessage(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			})

			require.NoError(t, err)
			assert.Equal(t, int64(1), view.Sequence)
		})

		t.Run("作成が競合しても先に作られた問い合わせへ追加する", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			i := newTestInquiry(t, userID)

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)
			d.repo.EXPECT().CreateIfAbsent(gomock.Any(), gomock.Any()).Return(i, nil)
			expectAppendSucceeds(t, d)

			_, err := u.AppendMessage(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			})

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("作成に失敗したら追加しない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			wantErr := xerrors.New("create failed")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)
			d.repo.EXPECT().CreateIfAbsent(gomock.Any(), gomock.Any()).Return(nil, wantErr)

			_, err := u.AppendMessage(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			})

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("問い合わせの取得に失敗したらそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			wantErr := xerrors.New("find failed")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, wantErr)

			_, err := u.AppendMessage(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			})

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func Test_usecase_appendForUser(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("送り手を利用者として追加する", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			i := newTestInquiry(t, userID)

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(i, nil)
			var created *domaininquiry.Message
			d.sequences.EXPECT().Allocate(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), nil).Times(2)
			d.repo.EXPECT().CreateMessage(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, m *domaininquiry.Message) error { created = m; return nil },
			)
			d.repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			d.repo.EXPECT().ListMessages(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]*domaininquiry.Message{newTestMessage(t, domaininquiry.AuthorKindUser, 1)}, nil)
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil).Times(2)

			_, err := u.appendForUser(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			})

			require.NoError(t, err)
			require.NotNil(t, created)
			assert.Equal(t, domaininquiry.AuthorKindUser, created.Author().Kind())
			assert.Equal(t, userID, created.Author().SubjectID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("問い合わせを決められなければ追加しない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			wantErr := xerrors.New("find failed")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, wantErr)

			_, err := u.appendForUser(context.Background(), AppendMessageParams{UserID: userID, Body: "本文"})

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func Test_usecase_resolveOrCreateInquiry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既にあればそれを返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			i := newTestInquiry(t, userID)

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(i, nil)

			got, err := u.resolveOrCreateInquiry(context.Background(), userID)

			require.NoError(t, err)
			assert.Equal(t, i, got)
		})

		t.Run("無ければ作成して返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			var created *domaininquiry.Inquiry
			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)
			d.repo.EXPECT().CreateIfAbsent(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, i *domaininquiry.Inquiry) (*domaininquiry.Inquiry, error) {
					created = i
					return i, nil
				},
			)

			got, err := u.resolveOrCreateInquiry(context.Background(), userID)

			require.NoError(t, err)
			assert.Equal(t, created, got)
			assert.Equal(t, userID, got.UserID())
		})

		t.Run("作成が競合したら先に作られたものが返る", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			i := newTestInquiry(t, userID)

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)
			d.repo.EXPECT().CreateIfAbsent(gomock.Any(), gomock.Any()).Return(i, nil)

			got, err := u.resolveOrCreateInquiry(context.Background(), userID)

			require.NoError(t, err)
			assert.Equal(t, i, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("競合以外の作成失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			wantErr := xerrors.New("create failed")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)
			d.repo.EXPECT().CreateIfAbsent(gomock.Any(), gomock.Any()).Return(nil, wantErr)

			_, err := u.resolveOrCreateInquiry(context.Background(), userID)

			require.ErrorIs(t, err, wantErr)
		})
	})
}
