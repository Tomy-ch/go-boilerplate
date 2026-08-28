package dynamodb

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// itemKind は、item の形が崩れていたときのエラー文に載せる種類です。
const itemKind = "instance lease"

// store は、realtime.InstanceLeaseStore の DynamoDB 実装です。時刻は比較を条件式で行うため
// epoch ナノ秒（N）で持ちます。
type store struct {
	c      *dynamodb.Client
	table  string
	tracer observability.LayerTracer
}

// New は、table を指す InstanceLeaseStore 実装を生成します。
func New(c *dynamodb.Client, table string, tf observability.TracerFactory) realtime.InstanceLeaseStore {
	return &store{c: c, table: table, tracer: tf.Infra()}
}

// Heartbeat は、lease を作成または更新します。回収の引き受け（cleanup_owner）には触れません。
func (s *store) Heartbeat(ctx context.Context, lease realtime.InstanceLease) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	if lease.InstanceID == "" {
		return xerrors.Wrap(apperror.ErrInvalidArgument, "heartbeat: instance id is empty")
	}

	_, err := s.c.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        aws.String(s.table),
		Key:              key(lease.InstanceID),
		UpdateExpression: aws.String("SET " + attrHeartbeatAt + " = :h, " + attrExpiresAt + " = :e"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":h": nano(lease.HeartbeatAt),
			":e": nano(lease.ExpiresAt),
		},
	})
	if err != nil {
		return dynamodbclient.Normalize(err, "heartbeat")
	}

	return nil
}

// Delete は、lease を削除します。無くても成功です。
func (s *store) Delete(ctx context.Context, id realtime.InstanceID) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	_, err := s.c.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(s.table), Key: key(id)})
	if err != nil {
		return dynamodbclient.Normalize(err, "delete lease")
	}

	return nil
}

// ListExpired は、asOf 時点で期限切れの lease を返します。instance の数は小さいので Scan で足ります。
func (s *store) ListExpired(ctx context.Context, asOf time.Time) ([]realtime.InstanceLease, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	var (
		leases []realtime.InstanceLease
		start  map[string]types.AttributeValue
	)

	for {
		out, err := s.c.Scan(ctx, &dynamodb.ScanInput{
			TableName:                 aws.String(s.table),
			FilterExpression:          aws.String(attrExpiresAt + " < :asOf"),
			ExpressionAttributeValues: map[string]types.AttributeValue{":asOf": nano(asOf)},
			ExclusiveStartKey:         start,
			ConsistentRead:            aws.Bool(true),
		})
		if err != nil {
			return nil, dynamodbclient.Normalize(err, "list expired leases")
		}

		for _, item := range out.Items {
			lease, err := fromItem(item)
			if err != nil {
				return nil, err
			}

			leases = append(leases, lease)
		}

		if len(out.LastEvaluatedKey) == 0 {
			return leases, nil
		}

		start = out.LastEvaluatedKey
	}
}

// AcquireCleanup は、lease が期限切れで未回収（または引き受けが失効）のときだけ引き受けを記録します。
// 条件式が成立しなければ他者が先に引き受けた（または lease が無い）ので false を返します。
func (s *store) AcquireCleanup(ctx context.Context, claim realtime.CleanupClaim) (bool, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	_, err := s.c.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        aws.String(s.table),
		Key:              key(claim.InstanceID),
		UpdateExpression: aws.String("SET " + attrCleanupOwner + " = :o, " + attrCleanupOwnerUntil + " = :u"),
		ConditionExpression: aws.String("attribute_exists(" + attrInstanceID + ") AND " + attrExpiresAt + " < :before AND " +
			"(attribute_not_exists(" + attrCleanupOwner + ") OR " + attrCleanupOwnerUntil + " < :now)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":o":      &types.AttributeValueMemberS{Value: claim.Owner},
			":u":      nano(claim.OwnerUntil),
			":before": nano(claim.ExpiredBefore),
			":now":    nano(claim.Now),
		},
	})
	if err == nil {
		return true, nil
	}

	if dynamodbclient.IsConditionalCheckFailed(err) {
		return false, nil
	}

	return false, dynamodbclient.Normalize(err, "acquire cleanup")
}

func key(id realtime.InstanceID) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{attrInstanceID: &types.AttributeValueMemberS{Value: string(id)}}
}

// nano は、時刻を epoch ナノ秒の N 属性にします。ゼロ値は 0 です。
func nano(t time.Time) *types.AttributeValueMemberN {
	if t.IsZero() {
		return &types.AttributeValueMemberN{Value: "0"}
	}

	return &types.AttributeValueMemberN{Value: strconv.FormatInt(t.UnixNano(), 10)}
}

// fromNano は、epoch ナノ秒を時刻に戻します。0 はゼロ値です。
func fromNano(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}

	return time.Unix(0, n).UTC()
}

// fromItem は、item を lease に戻します。回収の引き受けが無ければ CleanupOwner は空です。
func fromItem(item map[string]types.AttributeValue) (realtime.InstanceLease, error) {
	heartbeat, err := dynamodbclient.NumberAttr(item, attrHeartbeatAt, itemKind)
	if err != nil {
		return realtime.InstanceLease{}, err
	}

	expires, err := dynamodbclient.NumberAttr(item, attrExpiresAt, itemKind)
	if err != nil {
		return realtime.InstanceLease{}, err
	}

	lease := realtime.InstanceLease{
		InstanceID:   realtime.InstanceID(dynamodbclient.StringAttr(item, attrInstanceID)),
		HeartbeatAt:  fromNano(heartbeat),
		ExpiresAt:    fromNano(expires),
		CleanupOwner: dynamodbclient.StringAttr(item, attrCleanupOwner),
	}

	if _, has := item[attrCleanupOwnerUntil]; has {
		until, err := dynamodbclient.NumberAttr(item, attrCleanupOwnerUntil, itemKind)
		if err != nil {
			return realtime.InstanceLease{}, err
		}

		lease.CleanupOwnerUntil = fromNano(until)
	}

	return lease, nil
}
