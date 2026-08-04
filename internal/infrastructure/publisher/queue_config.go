package publisher

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/queue/sqs"
	"go-boilerplate/pkg/xerrors"
)

// ErrInvalidQueue は、SQS publish 先の設定が不正であることを示すエラーです。
var ErrInvalidQueue = xerrors.Wrap(apperror.ErrInvalidArgument, "invalid outbox queue")

// newQueueConfig は、config から SQS publisher の設定を解決して返します。
// 未設定のまま起動すると全 publish が失敗し、気付かぬうちに全メッセージが dead 化するため、
// HTTP 側の NewEndpoint と同じく relay 起動時点で弾きます（サイレント障害の防止）。
func newQueueConfig(cfg *config.OutboxConfig) (sqs.PublisherConfig, error) {
	if cfg.QueueURL() == "" {
		return sqs.PublisherConfig{}, xerrors.Wrap(ErrInvalidQueue, "OUTBOX_QUEUE_URL must not be empty")
	}
	// region は SigV4 署名に必須で、空のまま送ると署名不一致として送信時に落ちる。
	if cfg.QueueRegion() == "" {
		return sqs.PublisherConfig{}, xerrors.Wrap(ErrInvalidQueue, "OUTBOX_QUEUE_REGION must not be empty")
	}
	// 資格情報は静的注入のみを受け付ける。空でも SDK は署名を作ってしまい、認証エラーが publish 時まで
	// 顕在化しないため、ここで弾く。
	if cfg.QueueAccessKeyID() == "" || cfg.QueueSecretAccessKey() == "" {
		return sqs.PublisherConfig{}, xerrors.Wrap(
			ErrInvalidQueue, "OUTBOX_QUEUE_ACCESS_KEY_ID and OUTBOX_QUEUE_SECRET_ACCESS_KEY must not be empty")
	}
	return sqs.PublisherConfig{QueueURL: cfg.QueueURL()}, nil
}
