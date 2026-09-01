package inquiry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	domaininquiry "go-boilerplate/internal/domain/inquiry"
	domainmessage "go-boilerplate/internal/domain/inquirymessage"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"
)

// expectAppendSucceeds は、問い合わせが決まった後の追加経路が成功する期待を並べます。
func expectAppendSucceeds(t *testing.T, d deps, inquiryID uuid.UUID) {
	t.Helper()
	d.sequences.EXPECT().Allocate(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), nil).Times(2)
	d.messages.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	d.repo.EXPECT().Touch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	d.messages.EXPECT().ListByInquiry(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*domainmessage.Message{newTestMessage(t, inquiryID, domainmessage.AuthorKindUser, 1)}, nil)
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
			expectAppendSucceeds(t, d, i.ID())

			view, err := u.AppendMessage(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			})

			require.NoError(t, err)
			assert.Equal(t, int64(1), view.Sequence)
		})

		t.Run("作成が競合しても2回目で解決すれば成功する", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			i := newTestInquiry(t, userID)

			gomock.InOrder(
				d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound),
				d.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict),
				d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(i, nil),
			)
			expectAppendSucceeds(t, d, i.ID())

			_, err := u.AppendMessage(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			})

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("やり直しても問い合わせを決められなければConflictを返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			gomock.InOrder(
				d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound),
				d.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict),
				d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound),
			)

			_, err := u.AppendMessage(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			})

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("競合以外の失敗はやり直さずそのまま返す", func(t *testing.T) {
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
			var created *domainmessage.Message
			d.sequences.EXPECT().Allocate(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), nil).Times(2)
			d.messages.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, m *domainmessage.Message) error { created = m; return nil },
			)
			d.repo.EXPECT().Touch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.messages.EXPECT().ListByInquiry(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]*domainmessage.Message{newTestMessage(t, i.ID(), domainmessage.AuthorKindUser, 1)}, nil)
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil).Times(2)

			_, err := u.appendForUser(context.Background(), AppendMessageParams{
				UserID: userID, Subject: "user-john-doe", Body: "本文",
			}, true)

			require.NoError(t, err)
			require.NotNil(t, created)
			assert.Equal(t, domainmessage.AuthorKindUser, created.Author().Kind())
			assert.Equal(t, userID, created.Author().SubjectID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("問い合わせを決められなければ追加しない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)

			_, err := u.appendForUser(context.Background(), AppendMessageParams{UserID: userID, Body: "本文"}, false)

			require.ErrorIs(t, err, errInquiryCreationRace)
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

			got, err := u.resolveOrCreateInquiry(context.Background(), userID, true)

			require.NoError(t, err)
			assert.Equal(t, i, got)
		})

		t.Run("無ければ作成して返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			var created *domaininquiry.Inquiry
			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)
			d.repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, i *domaininquiry.Inquiry) error { created = i; return nil },
			)

			got, err := u.resolveOrCreateInquiry(context.Background(), userID, true)

			require.NoError(t, err)
			assert.Equal(t, created, got)
			assert.Equal(t, userID, got.UserID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("作成が許されていなければやり直しを求める", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)

			_, err := u.resolveOrCreateInquiry(context.Background(), userID, false)

			require.ErrorIs(t, err, errInquiryCreationRace)
		})

		t.Run("作成が競合したらやり直しを求める", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)
			d.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict)

			_, err := u.resolveOrCreateInquiry(context.Background(), userID, true)

			require.ErrorIs(t, err, errInquiryCreationRace)
		})

		t.Run("競合以外の作成失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			userID := uuidtestkit.NewTestFromSalt(t, "user")
			wantErr := xerrors.New("create failed")

			d.repo.EXPECT().FindActiveByUserID(gomock.Any(), userID).Return(nil, apperror.ErrNotFound)
			d.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(wantErr)

			_, err := u.resolveOrCreateInquiry(context.Background(), userID, true)

			require.ErrorIs(t, err, wantErr)
		})
	})
}
