package aws

import (
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	mock_aws "go-boilerplate/internal/infrastructure/realtime/aws/mock"
	"go-boilerplate/internal/observability"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
	"go-boilerplate/pkg/uuid"
)

func newPublisher(t *testing.T) (publisherbndry.Publisher, *mock_realtime.MockEventLogStore, *mock_aws.MockSNSAPI) {
	t.Helper()

	ctrl := gomock.NewController(t)
	log := mock_realtime.NewMockEventLogStore(ctrl)
	snsAPI := mock_aws.NewMockSNSAPI(ctrl)

	return NewPublisher(log, snsAPI, testTopicARN, observability.NewNoopTracerFactory(t)), log, snsAPI
}

// outboxMessage は、eventID の event を payload に持つ outbox message を返します。eventID が空なら payload の eventId を省きます。
func outboxMessage(t *testing.T, eventID string) (publisherbndry.Message, rt.DeliveryEvent) {
	t.Helper()

	id, err := uuid.New()
	require.NoError(t, err)

	event := rt.DeliveryEvent{
		EventID: eventID, StreamID: "stream-1", Sequence: 7, Type: "inquiry.message.appended.v1",
		OccurredAt: time.Date(
			2026,
			time.August,
			29,
			1,
			2,
			3,
			0,
			time.UTC,
		), SchemaVersion: 1, Payload: []byte(`{"body":"hi"}`),
	}
	payload, err := event.MarshalJSON()
	require.NoError(t, err)

	if eventID == "" {
		event.EventID = id.String()
	}

	return publisherbndry.Message{MessageID: id, EventType: event.Type, Payload: payload}, event
}

func TestNewPublisher(t *testing.T) {
	t.Parallel()

	p, _, _ := newPublisher(t)
	assert.NotNil(t, p)
}

func Test_publisher_Publish(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("EventLog へ append してから wakeup を publish する（eventId は message_id で埋まる）", func(t *testing.T) {
			t.Parallel()

			p, log, snsAPI := newPublisher(t)
			m, want := outboxMessage(t, "")
			gomock.InOrder(
				log.EXPECT().Append(gomock.Any(), want).Return(nil),
				snsAPI.EXPECT().Publish(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, in *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
						assert.Equal(t, testTopicARN, awssdk.ToString(in.TopicArn))
						assert.JSONEq(
							t,
							`{"eventId":"`+want.EventID+`","streamId":"stream-1","sequence":"7"}`,
							awssdk.ToString(in.Message),
						)
						assert.Equal(t, "wakeup", awssdk.ToString(in.MessageAttributes[AttrKind].StringValue))

						return &sns.PublishOutput{}, nil
					}),
			)

			require.NoError(t, p.Publish(t.Context(), m))
		})

		t.Run("append が冪等に成功する再実行でも wakeup を再 publish する（重複 wakeup は無害）", func(t *testing.T) {
			t.Parallel()

			p, log, snsAPI := newPublisher(t)
			m, _ := outboxMessage(t, "")
			log.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil).Times(2)
			snsAPI.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(&sns.PublishOutput{}, nil).Times(2)

			require.NoError(t, p.Publish(t.Context(), m))
			require.NoError(t, p.Publish(t.Context(), m))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("payload が復元できなければ ErrPermanent で、append も publish もしない", func(t *testing.T) {
			t.Parallel()

			p, _, _ := newPublisher(t)
			m, _ := outboxMessage(t, "")
			m.Payload = []byte("not json")

			require.ErrorIs(t, p.Publish(t.Context(), m), apperror.ErrPermanent)
		})

		t.Run("payload の eventId が message_id と食い違えば ErrEventIDMismatch（permanent）", func(t *testing.T) {
			t.Parallel()

			p, _, _ := newPublisher(t)
			m, _ := outboxMessage(t, "other-id")

			err := p.Publish(t.Context(), m)
			require.ErrorIs(t, err, ErrEventIDMismatch)
			require.ErrorIs(t, err, apperror.ErrPermanent)
		})

		t.Run("封筒が不正なら ErrPermanent", func(t *testing.T) {
			t.Parallel()

			p, _, _ := newPublisher(t)
			m, _ := outboxMessage(t, "")
			m.Payload = []byte(`{"eventId":"","streamId":"","sequence":"1"}`)

			require.ErrorIs(t, p.Publish(t.Context(), m), apperror.ErrPermanent)
		})

		t.Run("同じ位置に別の event があれば ErrPermanent で、publish しない", func(t *testing.T) {
			t.Parallel()

			p, log, _ := newPublisher(t)
			m, _ := outboxMessage(t, "")
			log.EXPECT().Append(gomock.Any(), gomock.Any()).Return(rt.ErrSequenceConflict)

			err := p.Publish(t.Context(), m)
			require.ErrorIs(t, err, apperror.ErrPermanent)
			require.ErrorIs(t, err, rt.ErrSequenceConflict)
		})

		t.Run("EventLog に届かなければ ErrRetryable で、publish しない", func(t *testing.T) {
			t.Parallel()

			p, log, _ := newPublisher(t)
			m, _ := outboxMessage(t, "")
			log.EXPECT().Append(gomock.Any(), gomock.Any()).Return(apperror.ErrUnavailable)

			err := p.Publish(t.Context(), m)
			require.ErrorIs(t, err, apperror.ErrRetryable)
			assert.NotErrorIs(t, err, apperror.ErrPermanent)
		})

		t.Run("append 済みで publish が失敗すれば ErrRetryable（再実行は append の冪等性で吸収される）", func(t *testing.T) {
			t.Parallel()

			p, log, snsAPI := newPublisher(t)
			m, _ := outboxMessage(t, "")
			log.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil)
			snsAPI.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			err := p.Publish(t.Context(), m)
			require.ErrorIs(t, err, apperror.ErrRetryable)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_decodeEvent(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("eventId が message_id と一致すればそのまま通る", func(t *testing.T) {
			t.Parallel()

			m, _ := outboxMessage(t, "")
			m2, _ := outboxMessage(t, m.MessageID.String())
			m2.MessageID = m.MessageID

			got, err := decodeEvent(m2)
			require.NoError(t, err)
			assert.Equal(t, m.MessageID.String(), got.EventID)
		})
	})
}

func Test_classifyAppend(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err       error
		permanent bool
	}{
		"位置の衝突は permanent":     {err: rt.ErrSequenceConflict, permanent: true},
		"不正な封筒は permanent":     {err: rt.ErrInvalidEvent, permanent: true},
		"大きすぎる封筒は permanent":   {err: rt.ErrPayloadTooLarge, permanent: true},
		"store の不通は retryable": {err: apperror.ErrUnavailable, permanent: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := classifyAppend(tt.err)
			require.ErrorIs(t, err, tt.err)
			if tt.permanent {
				require.ErrorIs(t, err, apperror.ErrPermanent)
			} else {
				require.ErrorIs(t, err, apperror.ErrRetryable)
			}
		})
	}
}
