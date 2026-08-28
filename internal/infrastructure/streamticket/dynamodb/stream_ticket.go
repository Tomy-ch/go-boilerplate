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
const itemKind = "stream ticket"

// subjectDestinationSeparator は、GSI の key に subject と destination を連結する区切りです。
// subject にも destination にも現れない文字を選びます。
const subjectDestinationSeparator = "\x1f"

// store は、realtime.StreamTicketStore の DynamoDB 実装です。
type store struct {
	c      *dynamodb.Client
	table  string
	tracer observability.LayerTracer
}

// New は、table を指す StreamTicketStore 実装を生成します。
func New(c *dynamodb.Client, table string, tf observability.TracerFactory) realtime.StreamTicketStore {
	return &store{c: c, table: table, tracer: tf.Infra()}
}

// Save は、ticket を hash を key として保存します（同じ hash は上書き）。
func (s *store) Save(ctx context.Context, ticket realtime.StreamTicket) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	if ticket.Hash == "" {
		return xerrors.Wrap(apperror.ErrInvalidArgument, "save ticket: hash is empty")
	}

	_, err := s.c.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(s.table), Item: toItem(ticket)})
	if err != nil {
		return dynamodbclient.Normalize(err, "save ticket")
	}

	return nil
}

// Find は、hash の ticket を強い一貫性で読みます。無い、または asOf 時点で期限切れなら ok=false です。
func (s *store) Find(ctx context.Context, hash realtime.TicketHash, asOf time.Time) (realtime.StreamTicket, bool, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	out, err := s.c.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(s.table),
		Key:            key(hash),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return realtime.StreamTicket{}, false, dynamodbclient.Normalize(err, "find ticket")
	}

	if len(out.Item) == 0 {
		return realtime.StreamTicket{}, false, nil
	}

	ticket, err := fromItem(out.Item)
	if err != nil {
		return realtime.StreamTicket{}, false, err
	}

	if !asOf.Before(ticket.ExpiresAt) {
		return realtime.StreamTicket{}, false, nil
	}

	return ticket, true, nil
}

// Invalidate は、subject × destination の ticket を GSI で引いてすべて削除します。
// GSI は結果整合なので、直前に Save した ticket が見えないことがあります。revocation の主機構は
// fan-out による接続の close で、ここは STOP を無視する client への保険です（設計正本 §2.4）。
func (s *store) Invalidate(ctx context.Context, subject string, destination realtime.StreamID) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	var start map[string]types.AttributeValue
	for {
		out, err := s.c.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(s.table),
			IndexName:              aws.String(indexBySubjectDestination),
			KeyConditionExpression: aws.String(attrSubjectDestination + " = :sd"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":sd": &types.AttributeValueMemberS{Value: subjectDestination(subject, destination)},
			},
			ExclusiveStartKey: start,
		})
		if err != nil {
			return dynamodbclient.Normalize(err, "invalidate tickets")
		}

		for _, item := range out.Items {
			hash, ok := item[attrTicketHash].(*types.AttributeValueMemberS)
			if !ok {
				return xerrors.Wrap(apperror.ErrInternal, "stream ticket index item: ticket_hash is not a string")
			}

			if _, err := s.c.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(s.table),
				Key:       key(realtime.TicketHash(hash.Value)),
			}); err != nil {
				return dynamodbclient.Normalize(err, "invalidate tickets")
			}
		}

		if len(out.LastEvaluatedKey) == 0 {
			return nil
		}

		start = out.LastEvaluatedKey
	}
}

func key(hash realtime.TicketHash) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{attrTicketHash: &types.AttributeValueMemberS{Value: string(hash)}}
}

// subjectDestination は、GSI の key を返します。
func subjectDestination(subject string, destination realtime.StreamID) string {
	return subject + subjectDestinationSeparator + string(destination)
}

// toItem は、ticket を item に写します。expires_at は epoch 秒（TTL 属性でもある）なので秒精度です。
func toItem(t realtime.StreamTicket) map[string]types.AttributeValue {
	item := key(t.Hash)
	item[attrSubject] = &types.AttributeValueMemberS{Value: t.Subject}
	item[attrDestination] = &types.AttributeValueMemberS{Value: string(t.Destination)}
	item[attrScope] = &types.AttributeValueMemberS{Value: t.Scope}
	item[attrInitialCursor] = &types.AttributeValueMemberN{Value: t.InitialCursor.String()}
	item[attrIssuedAt] = &types.AttributeValueMemberS{Value: t.IssuedAt.UTC().Format(time.RFC3339Nano)}
	item[attrExpiresAt] = &types.AttributeValueMemberN{Value: strconv.FormatInt(t.ExpiresAt.Unix(), 10)}
	item[attrSubjectDestination] = &types.AttributeValueMemberS{Value: subjectDestination(t.Subject, t.Destination)}

	return item
}

// fromItem は、item を ticket に戻します。
func fromItem(item map[string]types.AttributeValue) (realtime.StreamTicket, error) {
	cursor, err := dynamodbclient.NumberAttr(item, attrInitialCursor, itemKind)
	if err != nil {
		return realtime.StreamTicket{}, err
	}

	expires, err := dynamodbclient.NumberAttr(item, attrExpiresAt, itemKind)
	if err != nil {
		return realtime.StreamTicket{}, err
	}

	issuedAt, err := time.Parse(time.RFC3339Nano, dynamodbclient.StringAttr(item, attrIssuedAt))
	if err != nil {
		return realtime.StreamTicket{}, xerrors.Wrap(apperror.ErrInternal, "stream ticket item: issued_at: "+err.Error())
	}

	return realtime.StreamTicket{
		Hash:          realtime.TicketHash(dynamodbclient.StringAttr(item, attrTicketHash)),
		Subject:       dynamodbclient.StringAttr(item, attrSubject),
		Destination:   realtime.StreamID(dynamodbclient.StringAttr(item, attrDestination)),
		Scope:         dynamodbclient.StringAttr(item, attrScope),
		InitialCursor: realtime.Sequence(cursor),
		IssuedAt:      issuedAt,
		ExpiresAt:     time.Unix(expires, 0).UTC(),
	}, nil
}
