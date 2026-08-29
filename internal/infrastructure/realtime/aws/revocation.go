package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

var _ rt.RevocationNotifier = (*notifier)(nil)

// notifier は、rt.RevocationNotifier の SNS 実装です。wakeup と同じ topic に種別属性を変えて publish します。
type notifier struct {
	sns      SNSAPI
	topicARN string
	tracer   observability.LayerTracer
}

// NewRevocationNotifier は、topicARN へ失効通知を publish する RevocationNotifier を返します。
func NewRevocationNotifier(snsAPI SNSAPI, topicARN string, tf observability.TracerFactory) rt.RevocationNotifier {
	return &notifier{sns: snsAPI, topicARN: topicARN, tracer: tf.Infra()}
}

// NotifyRevoked は、subject × destination の失効を全 instance へ publish します。
func (n *notifier) NotifyRevoked(ctx context.Context, subject string, destination rt.StreamID) error {
	ctx, endSpan := n.tracer.Start(ctx)
	defer endSpan()

	body, attrs, err := encodeRevocation(rt.Revocation{Subject: subject, Destination: destination})
	if err != nil {
		return err
	}

	if _, err := n.sns.Publish(ctx, &sns.PublishInput{
		TopicArn:          awssdk.String(n.topicARN),
		Message:           awssdk.String(body),
		MessageAttributes: attrs,
	}); err != nil {
		return normalize(err, "publish revocation")
	}

	return nil
}
