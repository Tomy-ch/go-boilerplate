package sqs

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	mock_sqs "go-boilerplate/internal/infrastructure/queue/sqs/mock"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/publisher"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// spacedAuthorization は、前後空白付きの機微ヘッダ名です。map リテラルのキーに空白を書くと
// gocritic が誤記として弾くため、定数を経由して与えます。
const spacedAuthorization = " Authorization"

const testQueueURL = "http://elasticmq:9324/000000000000/gobp-events"

func newTestUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.New()
	require.NoError(t, err)
	return id
}

func newPublisher(t *testing.T, api API) boundary.Publisher {
	t.Helper()
	return NewPublisher(api, PublisherConfig{QueueURL: testQueueURL}, observability.NewNoopTracerFactory(t))
}

// captureSendMessage は、SendMessage の引数を捕捉するモックを仕込み、捕捉先を返します。
func captureSendMessage(t *testing.T, api *mock_sqs.MockAPI) **awssqs.SendMessageInput {
	t.Helper()
	captured := new(*awssqs.SendMessageInput)
	api.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in *awssqs.SendMessageInput, _ ...func(*awssqs.Options),
		) (*awssqs.SendMessageOutput, error) {
			*captured = in
			return &awssqs.SendMessageOutput{}, nil
		})
	return captured
}

func TestNewPublisher(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("API_Config_TracerFactory から Publisher を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)

			p := NewPublisher(api, PublisherConfig{QueueURL: testQueueURL}, observability.NewNoopTracerFactory(t))

			impl, ok := p.(*publisher)
			require.True(t, ok)
			assert.Equal(t, testQueueURL, impl.cfg.QueueURL)
		})
	})
}

func Test_publisher_Publish(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定したキューへ payload をそのまま本文として送る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			got := captureSendMessage(t, api)

			err := newPublisher(t, api).Publish(context.Background(), boundary.Message{
				MessageID: newTestUUID(t),
				EventType: "user.withdrawn.v1",
				Payload:   []byte(`{"userId":"u1"}`),
			})

			require.NoError(t, err)
			assert.Equal(t, testQueueURL, aws.ToString((*got).QueueUrl))
			assert.JSONEq(t, `{"userId":"u1"}`, aws.ToString((*got).MessageBody))
		})

		t.Run("組み立てた MessageAttributes を送信入力へ載せる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			got := captureSendMessage(t, api)
			messageID := newTestUUID(t)

			err := newPublisher(t, api).Publish(context.Background(), boundary.Message{
				MessageID: messageID,
				EventType: "user.withdrawn.v1",
			})

			require.NoError(t, err)
			assert.Equal(t, messageID.String(), aws.ToString((*got).MessageAttributes[AttrMessageID].StringValue))
		})

		t.Run("MessageAttributes が上限ちょうどなら送信する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			got := captureSendMessage(t, api)

			headers := make(map[string]string, maxMessageAttributes-reservedAttributes)
			for i := range maxMessageAttributes - reservedAttributes {
				headers[fmt.Sprintf("x-h%d", i)] = "v"
			}

			err := newPublisher(t, api).Publish(context.Background(), boundary.Message{
				MessageID: newTestUUID(t),
				EventType: "user.withdrawn.v1",
				Headers:   headers,
			})

			require.NoError(t, err)
			assert.Len(t, (*got).MessageAttributes, maxMessageAttributes)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SendMessage の失敗を ErrUnavailable へ正規化する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.New("broker down"))

			err := newPublisher(t, api).Publish(context.Background(), boundary.Message{
				MessageID: newTestUUID(t),
				EventType: "user.withdrawn.v1",
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("イベント種別が空なら送信前にエラーにする", func(t *testing.T) {
			t.Parallel()
			// SQS は空値の属性を拒むため、送っても必ず失敗する。relay が同じ行を延々と再送しないよう手前で弾く。
			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)

			err := newPublisher(t, api).Publish(context.Background(), boundary.Message{MessageID: newTestUUID(t)})

			require.ErrorIs(t, err, ErrMissingEventType)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("MessageAttributes が上限を超えたら送信前にエラーにする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)

			headers := make(map[string]string, maxMessageAttributes-reservedAttributes+1)
			for i := range maxMessageAttributes - reservedAttributes + 1 {
				headers[fmt.Sprintf("x-h%d", i)] = "v"
			}

			err := newPublisher(t, api).Publish(context.Background(), boundary.Message{
				MessageID: newTestUUID(t),
				EventType: "user.withdrawn.v1",
				Headers:   headers,
			})

			require.ErrorIs(t, err, ErrTooManyAttributes)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("ctx キャンセルを ErrCanceled へ正規化する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			api := mock_sqs.NewMockAPI(ctrl)
			api.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				Return(nil, context.Canceled)

			err := newPublisher(t, api).Publish(context.Background(), boundary.Message{
				MessageID: newTestUUID(t),
				EventType: "user.withdrawn.v1",
			})

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_publisher_messageAttributes(t *testing.T) {
	t.Parallel()

	p := &publisher{cfg: PublisherConfig{QueueURL: testQueueURL}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("outbox の message_id を String 型の属性として載せる", func(t *testing.T) {
			t.Parallel()

			messageID := newTestUUID(t)

			got := p.messageAttributes(boundary.Message{MessageID: messageID})

			assert.Equal(t, messageID.String(), aws.ToString(got[AttrMessageID].StringValue))
			assert.Equal(t, "String", aws.ToString(got[AttrMessageID].DataType))
		})

		t.Run("イベント種別を String 型の属性として載せる", func(t *testing.T) {
			t.Parallel()

			got := p.messageAttributes(boundary.Message{
				MessageID: newTestUUID(t),
				EventType: "user.withdrawn.v1",
			})

			assert.Equal(t, "user.withdrawn.v1", aws.ToString(got[AttrEventType].StringValue))
			assert.Equal(t, "String", aws.ToString(got[AttrEventType].DataType))
		})

		t.Run("伝搬対象ヘッダを属性として載せる", func(t *testing.T) {
			t.Parallel()

			got := p.messageAttributes(boundary.Message{
				MessageID: newTestUUID(t),
				Headers:   map[string]string{"traceparent": "00-trace-span-01"},
			})

			assert.Equal(t, "00-trace-span-01", aws.ToString(got["traceparent"].StringValue))
		})

		t.Run("機微ヘッダは前後空白付きでも載せない", func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				"Authorization":       "Bearer secret",
				"Proxy-Authorization": "Basic secret",
				"Cookie":              "session=secret",
				"Set-Cookie":          "session=secret",
			}
			headers[spacedAuthorization] = "Bearer secret"

			got := p.messageAttributes(boundary.Message{
				MessageID: newTestUUID(t),
				Headers:   headers,
			})

			assert.Len(t, got, reservedAttributes)
			assert.Contains(t, got, AttrMessageID)
			assert.Contains(t, got, AttrEventType)
		})

		t.Run("空値のヘッダは載せない", func(t *testing.T) {
			t.Parallel()

			got := p.messageAttributes(boundary.Message{
				MessageID: newTestUUID(t),
				Headers:   map[string]string{"empty": ""},
			})

			assert.NotContains(t, got, "empty")
		})

		t.Run("ヘッダは message_id を上書きできない", func(t *testing.T) {
			t.Parallel()

			messageID := newTestUUID(t)

			got := p.messageAttributes(boundary.Message{
				MessageID: messageID,
				Headers:   map[string]string{AttrMessageID: "spoofed"},
			})

			assert.Equal(t, messageID.String(), aws.ToString(got[AttrMessageID].StringValue))
		})

		t.Run("ヘッダは event_type を上書きできない", func(t *testing.T) {
			t.Parallel()

			got := p.messageAttributes(boundary.Message{
				MessageID: newTestUUID(t),
				EventType: "user.withdrawn.v1",
				Headers:   map[string]string{AttrEventType: "purchase.created.v1"},
			})

			assert.Equal(t, "user.withdrawn.v1", aws.ToString(got[AttrEventType].StringValue))
		})
	})
}
