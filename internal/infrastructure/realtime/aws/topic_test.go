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
)

func TestEnsureTopic(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("name で CreateTopic し、返った ARN をそのまま返す", func(t *testing.T) {
			t.Parallel()

			snsAPI := mock_aws.NewMockSNSAPI(gomock.NewController(t))
			snsAPI.EXPECT().CreateTopic(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ any, in *sns.CreateTopicInput, _ ...func(*sns.Options)) (*sns.CreateTopicOutput, error) {
					assert.Equal(t, "realtime-fanout-local", awssdk.ToString(in.Name))

					return &sns.CreateTopicOutput{TopicArn: awssdk.String(testTopicARN)}, nil
				})

			arn, err := EnsureTopic(t.Context(), snsAPI, "realtime-fanout-local")
			require.NoError(t, err)
			assert.Equal(t, testTopicARN, arn)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("substrate の失敗は ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			snsAPI := mock_aws.NewMockSNSAPI(gomock.NewController(t))
			snsAPI.EXPECT().CreateTopic(gomock.Any(), gomock.Any()).Return(nil, errSubstrate)

			_, err := EnsureTopic(t.Context(), snsAPI, "t")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("ARN が空で返れば ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			snsAPI := mock_aws.NewMockSNSAPI(gomock.NewController(t))
			snsAPI.EXPECT().CreateTopic(gomock.Any(), gomock.Any()).Return(&sns.CreateTopicOutput{}, nil)

			_, err := EnsureTopic(t.Context(), snsAPI, "t")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}
