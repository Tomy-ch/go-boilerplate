package module

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/worker/withdrawalarchive"
	"go-boilerplate/internal/infrastructure/queue/sqs"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	queuemetrics "go-boilerplate/internal/observability/metrics/queue"
	workerbd "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/pkg/xerrors"
)

// withdrawalArchiveAdapter は、退会証跡 worker が使う broker adapter の種別です（metric ラベル）。
const withdrawalArchiveAdapter = "sqs"

// ErrInvalidConsumerQueue は、consume 対象キューの設定が不正であることを示すエラーです。
var ErrInvalidConsumerQueue = xerrors.Wrap(apperror.ErrInvalidArgument, "invalid consumer queue")

// withdrawalArchiveQueue は、退会証跡 worker が使う broker クライアントと adapter 設定です。
// worker 本体と滞留量の収集対象が同じ接続を共有するよう、DI グラフ上で 1 度だけ解決します。
type withdrawalArchiveQueue struct {
	api sqs.API
	cfg sqs.Config
}

// provideWithdrawalArchiveQueue は、config から consume 側の SQS クライアントと adapter 設定を解決します。
// 未設定のまま起動すると受信が一度も成立せず、キューが溜まり続けていることに気付けないため、
// publish 側の newQueueConfig と同じく起動時点で弾きます。
func provideWithdrawalArchiveQueue(
	cfg *config.ConsumerQueueConfig, outbound *observability.OutboundHTTPClient,
) (withdrawalArchiveQueue, error) {
	if cfg.URL() == "" {
		return withdrawalArchiveQueue{}, xerrors.Wrap(ErrInvalidConsumerQueue, "CONSUMER_QUEUE_URL must not be empty")
	}
	// region は SigV4 署名に必須で、空のまま送ると署名不一致として受信時に落ちる。
	if cfg.Region() == "" {
		return withdrawalArchiveQueue{}, xerrors.Wrap(ErrInvalidConsumerQueue, "CONSUMER_QUEUE_REGION must not be empty")
	}

	api, err := sqs.NewClient(context.Background(), sqs.ClientConfig{
		Endpoint:        cfg.Endpoint(),
		Region:          cfg.Region(),
		AccessKeyID:     cfg.AccessKeyID(),
		SecretAccessKey: cfg.SecretAccessKey(),
		HTTPClient:      outbound,
	})
	if err != nil {
		return withdrawalArchiveQueue{}, err
	}

	return withdrawalArchiveQueue{
		api: api,
		cfg: sqs.Config{
			QueueURL:          cfg.URL(),
			DLQURL:            cfg.DLQURL(),
			MaxMessages:       cfg.MaxMessages(),
			WaitTimeSeconds:   cfg.WaitTimeSeconds(),
			VisibilityTimeout: cfg.VisibilityTimeout(),
		},
	}, nil
}

// provideWithdrawalArchiveWorker は、退会証跡 worker を組み立てます。
// broker adapter の生成をここに置くのは、controller 層が infrastructure を import できないためです。
func provideWithdrawalArchiveWorker(
	queue withdrawalArchiveQueue,
	archive user.ArchiveUsecase,
	tf observability.TracerFactory,
	logger logging.Logger,
) workerbd.Worker {
	// DLQ URL が無いまま退避先を配線すると、Permanent メッセージの退避が必ず失敗する。engine は
	// 退避に失敗したメッセージを Ack しないため、再配送で戻り続ける。退避先を持たない構成では
	// FailureHandler を配線せず、broker 側の redrive policy に委ねる。
	var failure workerbd.FailureHandler
	if queue.cfg.DLQURL != "" {
		failure = sqs.NewDeadLetter(queue.api, queue.cfg.DLQURL, tf)
	}

	return withdrawalarchive.New(
		sqs.NewConsumer(queue.api, queue.cfg, tf),
		failure,
		archive,
		tf,
		logger,
	)
}

// provideWithdrawalArchiveQueueStats は、退会証跡 worker のキュー滞留量の収集対象を組み立てます。
func provideWithdrawalArchiveQueueStats(
	queue withdrawalArchiveQueue, tf observability.TracerFactory,
) queuemetrics.Target {
	return queuemetrics.Target{
		WorkerName: withdrawalarchive.Name,
		Adapter:    withdrawalArchiveAdapter,
		Provider:   sqs.NewQueueStatsProvider(queue.api, queue.cfg, tf),
	}
}
