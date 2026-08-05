package publisher

import (
	"context" // sample-api:line

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/infrastructure/queue/sqs" // sample-api:line
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/publisher"
	"go-boilerplate/pkg/xerrors"
)

const (
	// KindHTTP は、受信エンドポイントへ HTTP POST する publish 先種別です。
	KindHTTP = "http"
	// sample-api:begin

	// KindSQS は、SQS 互換ブローカーへ送出する publish 先種別です。
	KindSQS = "sqs"
	// sample-api:end
)

// ErrUnknownKind は、判別子が未知の publish 先種別を指していることを示すエラーです。
var ErrUnknownKind = xerrors.Wrap(apperror.ErrInvalidArgument, "unknown outbox publisher kind")

// New は、判別子（OUTBOX_PUBLISHER）に対応する publish 実装を返します。
// publish 先の種別は環境ティアではなくデプロイ先の判断で決まるため、環境分岐ではなく明示の
// 判別子で選びます。対応する case が無い値は、意図しない publish 先へ黙って流れることを防ぐため
// 起動エラーにします（fail-closed）。
// sample-api:replace-begin
func New(
	cfg *config.OutboxConfig,
	client httpclient.Client,
	outbound *observability.OutboundHTTPClient,
	tf observability.TracerFactory,
) (boundary.Publisher, error) {
	// sample-api:replace-with
	// = func New(
	// = 	cfg *config.OutboxConfig,
	// = 	client httpclient.Client,
	// = 	tf observability.TracerFactory,
	// = ) (boundary.Publisher, error) {
	// sample-api:replace-end
	switch cfg.Publisher() {
	case KindHTTP:
		endpoint, err := NewEndpoint(cfg)
		if err != nil {
			return nil, err
		}
		return NewHTTP(endpoint, client, tf), nil
	// sample-api:begin
	case KindSQS:
		queueCfg, err := newQueueConfig(cfg)
		if err != nil {
			return nil, err
		}
		client, err := sqs.NewClient(context.Background(), sqs.ClientConfig{
			Endpoint:        cfg.QueueEndpoint(),
			Region:          cfg.QueueRegion(),
			AccessKeyID:     cfg.QueueAccessKeyID(),
			SecretAccessKey: cfg.QueueSecretAccessKey(),
			HTTPClient:      outbound,
		})
		if err != nil {
			return nil, err
		}
		return sqs.NewPublisher(client, queueCfg, tf), nil
	// sample-api:end
	default:
		// sample-api:replace-begin
		return nil, xerrors.Wrap(ErrUnknownKind, "OUTBOX_PUBLISHER must be one of: http, sqs")
		// sample-api:replace-with
		// = return nil, xerrors.Wrap(ErrUnknownKind, "OUTBOX_PUBLISHER must be one of: http")
		// sample-api:replace-end
	}
}
