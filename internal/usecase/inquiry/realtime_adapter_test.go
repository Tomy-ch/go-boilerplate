package inquiry

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domaininquiry "go-boilerplate/internal/domain/inquiry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
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
			m := newTestMessage(t, domaininquiry.AuthorKindUser, 3)

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
			m := newTestMessage(t, domaininquiry.AuthorKindUser, 3)

			var payloads [][]byte
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ucoutbox.EmitInput) (uuid.UUID, error) {
					payloads = append(payloads, in.Payload)
					return uuid.UUID{}, nil
				},
			).Times(2)

			require.NoError(t, u.emitDelivery(context.Background(), i, m, 9))

			require.Len(t, payloads, 2)
			feedEnvelope, err := rt.ParseDeliveryEvent(payloads[1])
			require.NoError(t, err)

			var feed map[string]any
			require.NoError(t, json.Unmarshal(feedEnvelope.Payload, &feed))
			assert.NotContains(t, feed, "body")
			assert.Equal(t, i.ID().String(), feed["inquiryId"])
		})

		t.Run("outboxへ載せるのは機構が読める配送封筒である", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			m := newTestMessage(t, domaininquiry.AuthorKindUser, 3)

			var payloads [][]byte
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ucoutbox.EmitInput) (uuid.UUID, error) {
					payloads = append(payloads, in.Payload)
					return uuid.UUID{}, nil
				},
			).Times(2)

			require.NoError(t, u.emitDelivery(context.Background(), i, m, 9))
			require.Len(t, payloads, 2)

			// feature の payload をそのまま載せると publish 側の ParseDeliveryEvent が読めず、
			// relay が恒久エラーとして dead 化する。両者を繋ぐのはこの直列化形だけなので、
			// 封筒として読めることと、機構が要る値が入っていることをここで固定する。
			conversation, err := rt.ParseDeliveryEvent(payloads[0])
			require.NoError(t, err)
			assert.Equal(t, rt.StreamID(i.ID().String()), conversation.StreamID)
			assert.Equal(t, rt.Sequence(m.Sequence()), conversation.Sequence)
			assert.Equal(t, event.TypeMessageCreated, conversation.Type)
			assert.Equal(t, event.SchemaVersionMessageCreated, conversation.SchemaVersion)
			assert.False(t, conversation.OccurredAt.IsZero())
			// EventID は outbox が採番する message_id と同じでなければならず、それが決まるのは
			// Emit の後。publish 側が埋める前提で空のまま出す。
			assert.Empty(t, conversation.EventID)

			feed, err := rt.ParseDeliveryEvent(payloads[1])
			require.NoError(t, err)
			assert.Equal(t, feedStreamID, feed.StreamID)
			// feed の位置は会話 stream とは別に採番される。取り違えると一覧の再開位置がずれる。
			assert.Equal(t, rt.Sequence(9), feed.Sequence)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("会話のemitが失敗したらfeedを出さない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			m := newTestMessage(t, domaininquiry.AuthorKindUser, 1)
			wantErr := xerrors.New("emit failed")

			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, wantErr)

			require.ErrorIs(t, u.emitDelivery(context.Background(), i, m, 9), wantErr)
		})

		t.Run("feedのemitが失敗したらそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			m := newTestMessage(t, domaininquiry.AuthorKindUser, 1)
			wantErr := xerrors.New("feed emit failed")

			gomock.InOrder(
				d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil),
				d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, wantErr),
			)

			require.ErrorIs(t, u.emitDelivery(context.Background(), i, m, 9), wantErr)
		})
	})
}

func Test_envelope(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("機構が読み戻せる直列化形を返す", func(t *testing.T) {
			t.Parallel()

			occurred := time.Date(2026, time.September, 3, 1, 2, 3, 0, time.UTC)
			b, err := envelope(rt.DeliveryEvent{
				StreamID: "stream-1", Sequence: 7, Type: "inquiry.message.created.v1",
				OccurredAt: occurred, SchemaVersion: 1, Payload: []byte(`{"k":"v"}`),
			})
			require.NoError(t, err)

			// publish 側が ParseDeliveryEvent で読む形であることが、両者を繋ぐ唯一の契約。
			got, err := rt.ParseDeliveryEvent(b)
			require.NoError(t, err)
			assert.Equal(t, rt.StreamID("stream-1"), got.StreamID)
			assert.Equal(t, rt.Sequence(7), got.Sequence)
			assert.Equal(t, "inquiry.message.created.v1", got.Type)
			assert.True(t, occurred.Equal(got.OccurredAt))
			assert.Equal(t, 1, got.SchemaVersion)
			assert.JSONEq(t, `{"k":"v"}`, string(got.Payload))
			// EventID は outbox が採番する message_id と等しくなければならず、決まるのは Emit の後。
			assert.Empty(t, got.EventID)
		})
	})
}
