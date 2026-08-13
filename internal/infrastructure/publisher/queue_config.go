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
// 未設定のまま起動すると relay 起動時点で弾きます（NewEndpoint と同じ理由。README.md 参照）。
func newQueueConfig(cfg *config.OutboxConfig) (sqs.PublisherConfig, error) {
	if cfg.QueueURL() == "" {
		return sqs.PublisherConfig{}, xerrors.Wrap(ErrInvalidQueue, "OUTBOX_QUEUE_URL must not be empty")
	}
	// region は SigV4 署名に必須で、空のまま送ると署名不一致として送信時に落ちる。
	if cfg.QueueRegion() == "" {
		return sqs.PublisherConfig{}, xerrors.Wrap(ErrInvalidQueue, "OUTBOX_QUEUE_REGION must not be empty")
	}
	// 資格情報はここで見ない。両方空は SDK 既定の credential chain（IAM ロール等）へ委ねる正当な指定で、
	// 解決できるかどうかは sqs.NewClient が起動時に確かめる。
	return sqs.PublisherConfig{QueueURL: cfg.QueueURL()}, nil
}
