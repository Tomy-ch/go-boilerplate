package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// EnsureTopic は、name の topic を実在させ、その ARN を返します。CreateTopic は既にある topic をそのまま返すので
// 何度呼んでも同じ状態に収束します。application の起動時ではなく one-shot の初期化（realtime-init）と contract test が呼びます。
func EnsureTopic(ctx context.Context, snsAPI SNSAPI, name string) (string, error) {
	out, err := snsAPI.CreateTopic(ctx, &sns.CreateTopicInput{Name: awssdk.String(name)})
	if err != nil {
		return "", normalize(err, "ensure topic")
	}

	arn := awssdk.ToString(out.TopicArn)
	if arn == "" {
		return "", xerrors.Wrap(apperror.ErrUnavailable, "ensure topic: empty TopicArn")
	}

	return arn, nil
}
