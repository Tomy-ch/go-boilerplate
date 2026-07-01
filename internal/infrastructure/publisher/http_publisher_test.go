package publisher_test

import (
	"context"
	"testing"

	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
	"go-boilerplate/internal/infrastructure/publisher"
	"go-boilerplate/internal/observability"
	pubbndry "go-boilerplate/internal/usecase/boundary/publisher"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testEndpoint = "https://receiver.example.com/events"

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Endpoint_Client_TracerFactory から Publisher を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)

			p := publisher.New(publisher.Endpoint(testEndpoint), client, observability.NewNoopTracerFactory(t))

			assert.NotNil(t, p)
		})
	})
}

func Test_httpPublisher_Publish(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("POST し 2xx で nil を返す。Idempotency-Key/Content-Type/headers を設定し retry は無効", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			msgID := uuid.NewTestFromSalt(t, "msg")

			client.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, req *httpclient.Request) (*httpclient.Response, error) {
					assert.Equal(t, httpclient.MethodPost(), req.Method())
					assert.Equal(t, testEndpoint, req.URL())
					assert.Equal(t, msgID.String(), req.IdempotencyKey())
					assert.False(t, req.AllowRetry())
					assert.Equal(t, []byte(`{"v":1}`), req.Body())
					assert.Equal(t, []string{"application/json"}, req.Header()["Content-Type"])
					assert.Equal(t, []string{"00-x"}, req.Header()["traceparent"])
					return &httpclient.Response{StatusCode: 200}, nil
				})

			err := publisher.New(publisher.Endpoint(testEndpoint), client, observability.NewNoopTracerFactory(t)).
				Publish(context.Background(), pubbndry.Message{
					MessageID: msgID,
					EventType: "e.v1",
					Payload:   []byte(`{"v":1}`),
					Headers:   map[string]string{"traceparent": "00-x"},
				})

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("client のエラー（非 2xx / transport 失敗）を伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			wantErr := xerrors.New("unavailable")

			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, wantErr)

			err := publisher.New(publisher.Endpoint(testEndpoint), client, observability.NewNoopTracerFactory(t)).
				Publish(context.Background(), pubbndry.Message{
					MessageID: uuid.NewTestFromSalt(t, "msg"),
					EventType: "e.v1",
					Payload:   []byte(`{}`),
				})

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func TestNewDownstreamProfile(t *testing.T) {
	t.Parallel()

	p := publisher.NewDownstreamProfile()

	// relay の poll ループが retry 本体のため transport retry は無効（MaxAttempts=1）。
	assert.Equal(t, 1, p.Profile.MaxAttempts)
	// traceparent は headers で明示伝搬するため自動 inject は抑止する。
	assert.False(t, p.Profile.PropagateTrace)
	// 送信先は外部エンドポイントのため private/loopback 宛ては拒否する。
	assert.False(t, p.Profile.AllowPrivateNetwork)
	assert.Equal(t, httpclient.Downstream("outbox"), p.Name)
}
