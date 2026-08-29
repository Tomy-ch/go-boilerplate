package realtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/infrastructure/awsclient"
	"go-boilerplate/internal/observability"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
)

const testTopicARN = "arn:aws:sns:us-east-1:000000000000:topic"

func testClients(t *testing.T) Clients {
	t.Helper()

	c, err := NewClients(t.Context(), ClientConfig{Endpoint: "http://localhost:4100", Region: "us-east-1", AccessKeyID: "k", SecretAccessKey: "s"})
	require.NoError(t, err)

	return c
}

func TestNewClients(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		c := testClients(t)
		assert.NotNil(t, c.SNS)
		assert.NotNil(t, c.SQS)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		_, err := NewClients(t.Context(), ClientConfig{Region: "us-east-1", SecretAccessKey: "s"})
		require.ErrorIs(t, err, awsclient.ErrInvalidCredentials)
	})
}

func TestNewPublisher(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	assert.NotNil(t, NewPublisher(mock_realtime.NewMockEventLogStore(ctrl), testClients(t), testTopicARN, observability.NewNoopTracerFactory(t)))
}

func TestNewRevocationNotifier(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewRevocationNotifier(testClients(t), testTopicARN, observability.NewNoopTracerFactory(t)))
}

func TestNewInstanceSubscription(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewInstanceSubscription(testClients(t), testTopicARN, "realtime-test", NewQueueAttributes(testTopicARN, ""), observability.NewNoopTracerFactory(t)))
}

func TestNewQueueAttributes(t *testing.T) {
	t.Parallel()

	attrs, err := NewQueueAttributes(testTopicARN, "arn:dlq").Build("arn:q")
	require.NoError(t, err)
	assert.Contains(t, attrs, "Policy")
	assert.Contains(t, attrs, "RedrivePolicy")
}

func TestNewEmulatorQueueAttributes(t *testing.T) {
	t.Parallel()

	attrs, err := NewEmulatorQueueAttributes().Build("arn:q")
	require.NoError(t, err)
	assert.NotContains(t, attrs, "Policy")
	assert.Contains(t, attrs, "VisibilityTimeout")
}
