package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/worker/withdrawalarchive"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_user "go-boilerplate/internal/usecase/user/mock"
)

const (
	testConsumerQueueURL    = "http://elasticmq:9324/000000000000/gobp-events"
	testConsumerQueueDLQURL = "http://elasticmq:9324/000000000000/gobp-events-dlq"
)

// newConsumerQueueConfig は、指定した URL / region で埋めた consumer キュー設定を返します。
func newConsumerQueueConfig(t *testing.T, url, region string) *config.ConsumerQueueConfig {
	t.Helper()

	cfg := config.NewConsumerQueueConfig(config.MockConfigForTest(t))
	cfg.SetConsumerQueue(t, url, testConsumerQueueDLQURL, region, "dummy-key", "dummy-secret")
	return cfg
}

// newWithdrawalArchiveQueueForTest は、正常な設定から解決した broker クライアントを返します。
func newWithdrawalArchiveQueueForTest(t *testing.T) withdrawalArchiveQueue {
	t.Helper()

	queue, err := provideWithdrawalArchiveQueue(
		newConsumerQueueConfig(t, testConsumerQueueURL, "us-east-1"),
		observability.NewDisabledOutboundHTTPClient(true),
	)
	require.NoError(t, err)
	return queue
}

func Test_provideWithdrawalArchiveQueue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("config を adapter 設定へ写す", func(t *testing.T) {
			t.Parallel()

			got, err := provideWithdrawalArchiveQueue(
				newConsumerQueueConfig(t, testConsumerQueueURL, "us-east-1"),
				observability.NewDisabledOutboundHTTPClient(true),
			)

			require.NoError(t, err)
			assert.NotNil(t, got.api)
			assert.Equal(t, testConsumerQueueURL, got.cfg.QueueURL)
			assert.Equal(t, testConsumerQueueDLQURL, got.cfg.DLQURL)
			// 受信条件は config の既定値がそのまま adapter へ渡ることを固定する。
			cfg := config.NewConsumerQueueConfig(config.MockConfigForTest(t))
			assert.Equal(t, cfg.MaxMessages(), got.cfg.MaxMessages)
			assert.Equal(t, cfg.WaitTimeSeconds(), got.cfg.WaitTimeSeconds)
			assert.Equal(t, cfg.VisibilityTimeout(), got.cfg.VisibilityTimeout)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キュー URL が空なら起動時点で弾く", func(t *testing.T) {
			t.Parallel()

			got, err := provideWithdrawalArchiveQueue(
				newConsumerQueueConfig(t, "", "us-east-1"),
				observability.NewDisabledOutboundHTTPClient(true),
			)

			require.ErrorIs(t, err, ErrInvalidConsumerQueue)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Nil(t, got.api)
		})

		t.Run("region が空なら起動時点で弾く", func(t *testing.T) {
			t.Parallel()
			// 空のまま署名すると不一致になり、受信時まで失敗が顕在化しない。
			got, err := provideWithdrawalArchiveQueue(
				newConsumerQueueConfig(t, testConsumerQueueURL, ""),
				observability.NewDisabledOutboundHTTPClient(true),
			)

			require.ErrorIs(t, err, ErrInvalidConsumerQueue)
			assert.Nil(t, got.api)
		})
	})
}

func Test_provideWithdrawalArchiveWorker(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("broker adapter を束ねた退会証跡 worker を返す", func(t *testing.T) {
			t.Parallel()

			got := provideWithdrawalArchiveWorker(
				newWithdrawalArchiveQueueForTest(t),
				mock_user.NewMockArchiveUsecase(gomock.NewController(t)),
				observability.NewNoopTracerFactory(t),
				logging.NewTestLogger(t),
			)

			assert.Equal(t, withdrawalarchive.Name, got.Name())
			assert.NotNil(t, got.Consumer())
			assert.NotNil(t, got.FailureHandler())
		})
	})
}

func Test_provideWithdrawalArchiveQueueStats(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker 名と adapter 種別を持つ収集対象を返す", func(t *testing.T) {
			t.Parallel()

			got := provideWithdrawalArchiveQueueStats(
				newWithdrawalArchiveQueueForTest(t),
				observability.NewNoopTracerFactory(t),
			)

			assert.Equal(t, withdrawalarchive.Name, got.WorkerName)
			assert.Equal(t, withdrawalArchiveAdapter, got.Adapter)
			assert.NotNil(t, got.Provider)
		})
	})
}
