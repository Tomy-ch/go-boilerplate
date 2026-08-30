package aws

import (
	"context"
	"slices"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

var _ rt.OrphanReclaimer = (*reclaimer)(nil)

// reclaimer は、rt.OrphanReclaimer の SNS / SQS 実装です。subscription は自分で組み立てず topic の
// 一覧から引き当てます（ARN の組み立ては account id / region の推測になり、emulator と production で
// 食い違う）。
type reclaimer struct {
	sns    SNSAPI
	sqs    SQSAPI
	target SubscriptionTarget
	tracer observability.LayerTracer
}

// NewOrphanReclaimer は、target の topic と queue 名の規則に対する OrphanReclaimer を返します。
func NewOrphanReclaimer(
	snsAPI SNSAPI, sqsAPI SQSAPI, target SubscriptionTarget, tf observability.TracerFactory,
) rt.OrphanReclaimer {
	return &reclaimer{sns: snsAPI, sqs: sqsAPI, target: target, tracer: tf.Infra()}
}

// Reclaim は、id の instance が残した登録を解除してから受信先を削除します。順序は固定です
// （先に受信先を消すと、登録が宛先の無いまま topic に残る）。既に無いものは何もしません。
// 片方が失敗しても残りを試み、失敗をまとめて返します。
//
// 削除する名前は topic の登録の endpoint から引き当て、設定から導ける名前も候補に足します。
// 片方だけでは届かない残骸がそれぞれにあります（README の Port mapping / Reclaim）。
func (r *reclaimer) Reclaim(ctx context.Context, id rt.InstanceID) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	configured, err := QueueName(r.target.QueuePrefix, id)
	if err != nil {
		return err
	}

	var errs []error

	found, err := r.unsubscribeAll(ctx, id)
	if err != nil {
		errs = append(errs, err)
	}

	for _, name := range candidateQueues(configured, found) {
		if err := r.deleteQueue(ctx, name); err != nil {
			errs = append(errs, err)
		}
	}

	return xerrors.Join(errs...)
}

// unsubscribeAll は、id の instance の受信先を指す登録をすべて解除し、その受信先の名前を返します。
// 同じ受信先に複数の登録がぶら下がる状態（Provision の途中失敗の残骸）も畳みます。
func (r *reclaimer) unsubscribeAll(ctx context.Context, id rt.InstanceID) ([]string, error) {
	var (
		errs  []error
		found []string
		token *string
	)

	for {
		out, err := r.sns.ListSubscriptionsByTopic(ctx, &sns.ListSubscriptionsByTopicInput{
			TopicArn:  awssdk.String(r.target.TopicARN),
			NextToken: token,
		})
		if err != nil {
			// 途中のページで止まっても、それまでに解除できなかった分は診断に要る。
			return found, xerrors.Join(append(errs, normalize(err, "list subscriptions by topic"))...)
		}

		for _, s := range out.Subscriptions {
			name := queueNameFromEndpoint(awssdk.ToString(s.Endpoint))
			if !queueBelongsTo(name, id) {
				continue
			}

			found = append(found, name)

			arn := awssdk.ToString(s.SubscriptionArn)
			if !isUnsubscribable(arn) {
				continue
			}

			if _, err := r.sns.Unsubscribe(ctx, &sns.UnsubscribeInput{SubscriptionArn: awssdk.String(arn)}); err != nil {
				errs = append(errs, normalize(err, "unsubscribe orphan queue"))
			}
		}

		token = out.NextToken
		if token == nil {
			return found, xerrors.Join(errs...)
		}
	}
}

// deleteQueue は、name の受信先を削除します。既に無ければ何もしません。
func (r *reclaimer) deleteQueue(ctx context.Context, name string) error {
	out, err := r.sqs.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: awssdk.String(name)})
	if err != nil {
		if queueGone(err) {
			return nil
		}

		return normalize(err, "resolve orphan queue url")
	}

	if _, err := r.sqs.DeleteQueue(ctx, &sqs.DeleteQueueInput{QueueUrl: out.QueueUrl}); err != nil {
		if queueGone(err) {
			return nil
		}

		return normalize(err, "delete orphan queue")
	}

	return nil
}

// candidateQueues は、削除を試みる受信先の名前を重複なく昇順で返します。
func candidateQueues(configured string, found []string) []string {
	names := append(slices.Clone(found), configured)
	slices.Sort(names)

	return slices.Compact(names)
}

// queueNameFromEndpoint は、登録の endpoint（queue ARN）から受信先の名前を返します。ARN の末尾要素が
// 名前で、名前自体に `:` は使えません。引けなければ空を返します。
func queueNameFromEndpoint(endpoint string) string {
	i := strings.LastIndex(endpoint, ":")
	if i < 0 || i == len(endpoint)-1 {
		return ""
	}

	return endpoint[i+1:]
}

// queueBelongsTo は、受信先の名前（`<prefix>-<instance id>`）が id の instance のものかを返します。
// prefix でなく識別子の側で引き当てます（旧 prefix の残骸にも届かせるため。README の Reclaim）。
func queueBelongsTo(name string, id rt.InstanceID) bool {
	return name != "" && strings.HasSuffix(name, "-"+string(id))
}

// isUnsubscribable は、SubscriptionArn が解除できる値かを返します。確認待ちの登録は ARN の代わりに
// "PendingConfirmation" を返し、それを渡すと Unsubscribe は失敗します。
func isUnsubscribable(arn string) bool {
	return arn != "" && strings.HasPrefix(arn, "arn:")
}

// queueGone は、受信先が既に無いことを示す失敗かを返します。成功と扱うか作り直しの契機と扱うかは呼び出し側が決めます。
func queueGone(err error) bool {
	var notExist *sqstypes.QueueDoesNotExist

	return xerrors.As(err, &notExist)
}
