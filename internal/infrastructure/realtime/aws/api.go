//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package aws は、Realtime Delivery の fan-out substrate（SNS topic → serve instance ごとの SQS queue）の
// AWS SDK v2 実装です。publish 側（EventLog へ append してから wakeup を publish する outbox publisher と、
// 失効通知の RevocationNotifier）と、受信側（instance 固有の queue と subscription の lifecycle）を持ちます。
// wakeup は状態を運ばず、重複は同じ読み直しに畳まれ、欠落は periodic catch-up が覆います（ADR-0073）。
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// SNSAPI は、この package が使う SNS の操作だけを抽象化したものです（*sns.Client が満たします）。
type SNSAPI interface {
	Publish(ctx context.Context, in *sns.PublishInput, opts ...func(*sns.Options)) (*sns.PublishOutput, error)
	Subscribe(ctx context.Context, in *sns.SubscribeInput, opts ...func(*sns.Options)) (*sns.SubscribeOutput, error)
	SetSubscriptionAttributes(
		ctx context.Context,
		in *sns.SetSubscriptionAttributesInput,
		opts ...func(*sns.Options),
	) (*sns.SetSubscriptionAttributesOutput, error)
	Unsubscribe(ctx context.Context, in *sns.UnsubscribeInput, opts ...func(*sns.Options)) (*sns.UnsubscribeOutput, error)
}

// SQSAPI は、この package が使う SQS の操作だけを抽象化したものです（*sqs.Client が満たします）。
type SQSAPI interface {
	CreateQueue(ctx context.Context, in *sqs.CreateQueueInput, opts ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error)
	GetQueueAttributes(
		ctx context.Context,
		in *sqs.GetQueueAttributesInput,
		opts ...func(*sqs.Options),
	) (*sqs.GetQueueAttributesOutput, error)
	SetQueueAttributes(
		ctx context.Context,
		in *sqs.SetQueueAttributesInput,
		opts ...func(*sqs.Options),
	) (*sqs.SetQueueAttributesOutput, error)
	ReceiveMessage(ctx context.Context, in *sqs.ReceiveMessageInput, opts ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, in *sqs.DeleteMessageInput, opts ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	DeleteQueue(ctx context.Context, in *sqs.DeleteQueueInput, opts ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error)
}
