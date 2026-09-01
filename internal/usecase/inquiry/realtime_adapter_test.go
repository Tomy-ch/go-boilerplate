package inquiry

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domainmessage "go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/usecase/inquiry/event"
	ucoutbox "go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"
)

func Test_conversationStreamID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("問い合わせIDをそのままstreamの識別子にする", func(t *testing.T) {
			t.Parallel()
			id := uuidtestkit.NewTestFromSalt(t, "inquiry").String()

			assert.Equal(t, id, conversationStreamID(id).String())
		})
	})
}

func Test_usecase_emitDelivery(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("会話とfeedへ順に1件ずつ出す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			m := newTestMessage(t, i.ID(), domainmessage.AuthorKindUser, 3)

			var emitted []ucoutbox.EmitInput
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ucoutbox.EmitInput) (uuid.UUID, error) {
					emitted = append(emitted, in)
					return uuid.UUID{}, nil
				},
			).Times(2)

			require.NoError(t, u.emitDelivery(context.Background(), i, m, 9))

			require.Len(t, emitted, 2)
			assert.Equal(t, event.TypeMessageCreated, emitted[0].EventType)
			assert.Equal(t, event.TypeThreadUpdated, emitted[1].EventType)
			assert.Equal(t, aggregateType, emitted[0].AggregateType)
			assert.Equal(t, i.ID().String(), emitted[0].AggregateID)
			// feed 側も同じ集約を指す。ここを見ないと、片方だけ別 ID へ差し替えても緑のまま通る。
			assert.Equal(t, aggregateType, emitted[1].AggregateType)
			assert.Equal(t, i.ID().String(), emitted[1].AggregateID)
		})

		t.Run("feedのeventは本文を持たない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			m := newTestMessage(t, i.ID(), domainmessage.AuthorKindUser, 3)

			var payloads [][]byte
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ucoutbox.EmitInput) (uuid.UUID, error) {
					payloads = append(payloads, in.Payload)
					return uuid.UUID{}, nil
				},
			).Times(2)

			require.NoError(t, u.emitDelivery(context.Background(), i, m, 9))

			require.Len(t, payloads, 2)
			var feed map[string]any
			require.NoError(t, json.Unmarshal(payloads[1], &feed))
			assert.NotContains(t, feed, "body")
			assert.Equal(t, i.ID().String(), feed["inquiryId"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("会話のemitが失敗したらfeedを出さない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			m := newTestMessage(t, i.ID(), domainmessage.AuthorKindUser, 1)
			wantErr := xerrors.New("emit failed")

			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, wantErr)

			require.ErrorIs(t, u.emitDelivery(context.Background(), i, m, 9), wantErr)
		})

		t.Run("feedのemitが失敗したらそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			m := newTestMessage(t, i.ID(), domainmessage.AuthorKindUser, 1)
			wantErr := xerrors.New("feed emit failed")

			gomock.InOrder(
				d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil),
				d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, wantErr),
			)

			require.ErrorIs(t, u.emitDelivery(context.Background(), i, m, 9), wantErr)
		})
	})
}
