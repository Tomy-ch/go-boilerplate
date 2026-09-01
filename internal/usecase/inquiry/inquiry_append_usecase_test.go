package inquiry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	domainmessage "go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/usecase/boundary/outbox"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucoutbox "go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"
)

// newTestUserAuthor は、利用者としての送り手を組み立てます。
func newTestUserAuthor(t *testing.T) domainmessage.Author {
	t.Helper()
	author, err := domainmessage.NewAuthor(
		domainmessage.AuthorKindUser, uuidtestkit.NewTestFromSalt(t, "subject"),
	)
	require.NoError(t, err)
	return author
}

func Test_usecase_appendMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("採番から2行のemitまでを順に行い読み直した内容を返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			stored := newTestMessage(t, i.ID(), domainmessage.AuthorKindUser, 3)

			var emitted []ucoutbox.EmitInput
			gomock.InOrder(
				d.sequences.EXPECT().Allocate(gomock.Any(), rt.StreamID(i.ID().String())).Return(rt.Sequence(3), nil),
				d.messages.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil),
				d.repo.EXPECT().Touch(gomock.Any(), i.ID(), baseTime).Return(nil),
				d.sequences.EXPECT().Allocate(gomock.Any(), feedStreamID).Return(rt.Sequence(9), nil),
				d.messages.EXPECT().ListByInquiry(gomock.Any(), i.ID(), gomock.Any()).
					Return([]*domainmessage.Message{stored}, nil),
			)
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ucoutbox.EmitInput) (uuid.UUID, error) {
					emitted = append(emitted, in)
					return uuidtestkit.NewTestFromSalt(t, "outbox"), nil
				},
			).Times(2)

			view, err := u.appendMessage(context.Background(), i, newTestUserAuthor(t), "本文")

			require.NoError(t, err)
			assert.Equal(t, int64(3), view.Sequence)
			require.Len(t, emitted, 2)
			assert.Equal(t, i.ID().String(), emitted[0].OrderingKey)
			assert.Equal(t, int64(3), emitted[0].OrderingSequence)
			assert.Equal(t, feedStreamID.String(), emitted[1].OrderingKey)
			assert.Equal(t, int64(9), emitted[1].OrderingSequence)
			assert.Equal(t, outbox.ChannelRealtime, emitted[0].Channel)
			assert.Equal(t, outbox.ChannelRealtime, emitted[1].Channel)
		})

		t.Run("更新日時は時計の値で進む", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			stored := newTestMessage(t, i.ID(), domainmessage.AuthorKindUser, 1)

			d.sequences.EXPECT().Allocate(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), nil).Times(2)
			d.messages.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			d.repo.EXPECT().Touch(gomock.Any(), i.ID(), baseTime).Return(nil)
			d.messages.EXPECT().ListByInquiry(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]*domainmessage.Message{stored}, nil)
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil).Times(2)

			_, err := u.appendMessage(context.Background(), i, newTestUserAuthor(t), "本文")

			require.NoError(t, err)
			assert.Equal(t, baseTime, i.UpdatedAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("会話streamの採番に失敗したら追加しない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			wantErr := xerrors.New("allocate failed")

			d.sequences.EXPECT().Allocate(gomock.Any(), gomock.Any()).Return(rt.Sequence(0), wantErr)

			_, err := u.appendMessage(context.Background(), i, newTestUserAuthor(t), "本文")

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("本文が不正なら追加しない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))

			d.sequences.EXPECT().Allocate(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), nil)

			_, err := u.appendMessage(context.Background(), i, newTestUserAuthor(t), "")

			require.ErrorIs(t, err, domainmessage.ErrEmptyBody)
		})

		t.Run("feed streamの採番に失敗したらemitしない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			wantErr := xerrors.New("feed allocate failed")

			d.sequences.EXPECT().Allocate(gomock.Any(), rt.StreamID(i.ID().String())).Return(rt.Sequence(1), nil)
			d.messages.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			d.repo.EXPECT().Touch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.sequences.EXPECT().Allocate(gomock.Any(), feedStreamID).Return(rt.Sequence(0), wantErr)

			_, err := u.appendMessage(context.Background(), i, newTestUserAuthor(t), "本文")

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func Test_usecase_readBackMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("直前の位置より後ろを1件だけ読み直す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			inquiryID := uuidtestkit.NewTestFromSalt(t, "inquiry")
			stored := newTestMessage(t, inquiryID, domainmessage.AuthorKindUser, 5)

			var captured domainmessage.HistoryParams
			d.messages.EXPECT().ListByInquiry(gomock.Any(), inquiryID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, params domainmessage.HistoryParams) ([]*domainmessage.Message, error) {
					captured = params
					return []*domainmessage.Message{stored}, nil
				},
			)

			got, err := u.readBackMessage(context.Background(), inquiryID, 5)

			require.NoError(t, err)
			assert.Equal(t, stored, got)
			require.NotNil(t, captured.AfterSequence)
			assert.Equal(t, int64(4), *captured.AfterSequence)
			assert.Equal(t, int64(5), captured.UpToSequence)
			assert.Equal(t, 1, captured.Limit)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読み直せなければ内部エラーを返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			d.messages.EXPECT().ListByInquiry(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]*domainmessage.Message{}, nil)

			_, err := u.readBackMessage(context.Background(), uuidtestkit.NewTestFromSalt(t, "inquiry"), 1)

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("読み出しが失敗したらそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			wantErr := xerrors.New("list failed")

			d.messages.EXPECT().ListByInquiry(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, wantErr)

			_, err := u.readBackMessage(context.Background(), uuidtestkit.NewTestFromSalt(t, "inquiry"), 1)

			require.ErrorIs(t, err, wantErr)
		})
	})
}
