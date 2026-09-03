// Package realtime は、Realtime Delivery の fan-out substrate（wakeup の publish と、serve instance ごとの
// 受信先）の実装を選ぶ唯一の場所です。背後の substrate を差し替える場合に書き換えるのはこのパッケージだけで、
// DI はここを通ります。
package realtime

import (
	"context"

	"go-boilerplate/internal/infrastructure/realtime/aws"
	"go-boilerplate/internal/infrastructure/realtime/local"
	"go-boilerplate/internal/observability"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// ClientConfig は、fan-out substrate のクライアント設定です。
type ClientConfig = aws.ClientConfig

// Clients は、同じ資格情報で組み立てた SNS / SQS クライアントの組です。
type Clients = aws.Clients

// AttributesBuilder は、instance queue に設定する属性の集合を組み立てる境界です。
type AttributesBuilder = aws.AttributesBuilder

// QueueAttributesInput は、production の属性を組み立てるのに要る deployment 依存の識別子です。
type QueueAttributesInput = aws.QueueAttributesInput

// SubscriptionTarget は、instance の受信先をどの topic に、どの名前で作るかです。
type SubscriptionTarget = aws.SubscriptionTarget

// NewClients は、設定から SNS / SQS クライアントを生成します。
func NewClients(ctx context.Context, cfg ClientConfig) (Clients, error) {
	return aws.NewClients(ctx, cfg)
}

// NewPublisher は、realtime channel の outbox publisher（EventLog へ append → wakeup を publish）を返します。
func NewPublisher(
	log rt.EventLogStore,
	c Clients,
	topicARN string,
	tf observability.TracerFactory,
	metrics *observability.RealtimeMetrics,
) publisherbndry.Publisher {
	return aws.NewPublisher(log, c.SNS, topicARN, tf, metrics)
}

// NewRevocationNotifier は、失効通知を全 instance へ publish する RevocationNotifier を返します。
func NewRevocationNotifier(c Clients, topicARN string, tf observability.TracerFactory) rt.RevocationNotifier {
	return aws.NewRevocationNotifier(c.SNS, topicARN, tf)
}

// NewInstanceSubscription は、instance 固有の受信先（queue + subscription）の lifecycle を持つ InstanceSubscription を返します。
func NewInstanceSubscription(
	c Clients, target SubscriptionTarget, attrs AttributesBuilder, tf observability.TracerFactory,
) rt.InstanceSubscription {
	return aws.NewInstanceSubscription(c.SNS, c.SQS, target, attrs, tf)
}

// NewOrphanReclaimer は、死んだ instance が残した受信先を識別子から辿って片付ける OrphanReclaimer を返します。
func NewOrphanReclaimer(c Clients, target SubscriptionTarget, tf observability.TracerFactory) rt.OrphanReclaimer {
	return aws.NewOrphanReclaimer(c.SNS, c.SQS, target, tf)
}

// EnsureTopic は、name の topic を実在させ、その ARN を返します（one-shot の初期化と contract test 用。application の起動時には呼ばない）。
func EnsureTopic(ctx context.Context, c Clients, name string) (string, error) {
	return aws.EnsureTopic(ctx, c.SNS, name)
}

// NewQueueAttributes は、production の instance queue の属性（policy / redrive / 暗号化 / timings）を返します。
func NewQueueAttributes(in QueueAttributesInput) AttributesBuilder {
	return aws.NewQueueAttributes(in)
}

// NewEmulatorQueueAttributes は、emulator が受け付ける属性（timings）だけを返します。
func NewEmulatorQueueAttributes() AttributesBuilder {
	return local.NewQueueAttributes()
}
