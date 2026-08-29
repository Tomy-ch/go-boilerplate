package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	mock_aws "go-boilerplate/internal/infrastructure/realtime/aws/mock"
	"go-boilerplate/internal/observability"
)

func TestNewRevocationNotifier(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	assert.NotNil(t, NewRevocationNotifier(mock_aws.NewMockSNSAPI(ctrl), testTopicARN, observability.NewNoopTracerFactory(t)))
}

func Test_notifier_NotifyRevoked(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失効通知を revocation 属性付きで topic へ publish する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			snsAPI := mock_aws.NewMockSNSAPI(ctrl)
			snsAPI.EXPECT().Publish(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ any, in *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
					assert.Equal(t, testTopicARN, awssdk.ToString(in.TopicArn))
					assert.JSONEq(t, `{"subject":"u1","destination":"s1"}`, awssdk.ToString(in.Message))
					assert.Equal(t, "revocation", awssdk.ToString(in.MessageAttributes[AttrKind].StringValue))

					return &sns.PublishOutput{}, nil
				})

			n := NewRevocationNotifier(snsAPI, testTopicARN, observability.NewNoopTracerFactory(t))
			require.NoError(t, n.NotifyRevoked(t.Context(), "u1", "s1"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("substrate の失敗は ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			snsAPI := mock_aws.NewMockSNSAPI(ctrl)
			snsAPI.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			n := NewRevocationNotifier(snsAPI, testTopicARN, observability.NewNoopTracerFactory(t))
			require.ErrorIs(t, n.NotifyRevoked(t.Context(), "u1", "s1"), apperror.ErrUnavailable)
		})
	})
}
