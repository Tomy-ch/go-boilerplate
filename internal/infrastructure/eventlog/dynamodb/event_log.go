package dynamodb

import (
	"context"
	"encoding/json"
	"math"
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

const (
	// itemKind は、item の形が崩れていたときのエラー文に載せる種類です。
	itemKind = "event log"
	// defaultReadLimit は、ReadAfter の Limit が 0 以下のときの 1 回の件数です。
	defaultReadLimit = 100
	// maxReadLimit は、1 回の Query に渡す件数の上限です。DynamoDB は 1 MiB で打ち切るので、
	// これを超えても HasMore で続きを読むだけで意味が変わりません。
	maxReadLimit = 1000
)

// store は、realtime.EventLogStore の DynamoDB 実装です。
type store struct {
	c      *dynamodb.Client
	table  string
	tracer observability.LayerTracer
}

// New は、table を指す EventLogStore 実装を生成します。
func New(c *dynamodb.Client, table string, tf observability.TracerFactory) realtime.EventLogStore {
	return &store{c: c, table: table, tracer: tf.Infra()}
}

// Append は、event を (StreamID, Sequence) の位置へ条件付きで書きます。位置が埋まっていたら
// 既存 item の EventID を読み比べ、同じなら成功（冪等）、違えば ErrSequenceConflict を返します。
func (s *store) Append(ctx context.Context, event realtime.DeliveryEvent) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	if err := event.Validate(); err != nil {
		return err
	}

	_, err := s.c.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.table),
		Item:                toItem(event),
		ConditionExpression: aws.String("attribute_not_exists(" + attrStreamID + ")"),
	})
	if err == nil {
		return s.advanceWatermark(ctx, event.StreamID, event.Sequence)
	}

	if !dynamodbclient.IsConditionalCheckFailed(err) {
		return dynamodbclient.Normalize(err, "append event")
	}

	existing, ok, err := s.Find(ctx, event.StreamID, event.Sequence)
	if err != nil {
		return err
	}

	if !ok {
		return xerrors.Wrap(apperror.ErrUnavailable, "append event: position was taken but the item could not be read back")
	}

	if existing.EventID != event.EventID {
		return xerrors.Wrap(realtime.ErrSequenceConflict, "stream "+string(event.StreamID)+" sequence "+event.Sequence.String())
	}

	return s.advanceWatermark(ctx, event.StreamID, event.Sequence)
}

// advanceWatermark は、追記済みの位置を seq まで進めます。後戻りさせないため、既存が seq 以上なら
// 書き換えません。event を書いた後に呼ぶので、ここで失敗しても watermark が event を追い越しません
// （呼出側の retry と Append の冪等性で埋まります）。
func (s *store) advanceWatermark(ctx context.Context, streamID realtime.StreamID, seq realtime.Sequence) error {
	_, err := s.c.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:           aws.String(s.table),
		Key:                 watermarkKey(streamID),
		UpdateExpression:    aws.String("SET #w = :seq"),
		ConditionExpression: aws.String("attribute_not_exists(#w) OR #w < :seq"),
		ExpressionAttributeNames: map[string]string{
			"#w": attrAppendedThrough,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":seq": &types.AttributeValueMemberN{Value: seq.String()},
		},
	})
	if err != nil && !dynamodbclient.IsConditionalCheckFailed(err) {
		return dynamodbclient.Normalize(err, "advance the append watermark")
	}

	return nil
}

// AppendedThrough は、この stream へ追記した最大の位置を返します。1 度も追記していなければ 0 です。
func (s *store) AppendedThrough(ctx context.Context, streamID realtime.StreamID) (realtime.Sequence, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	out, err := s.c.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(s.table),
		Key:            watermarkKey(streamID),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return 0, dynamodbclient.Normalize(err, "read the append watermark")
	}

	if len(out.Item) == 0 {
		return 0, nil
	}

	seq, err := dynamodbclient.NumberAttr(out.Item, attrAppendedThrough, itemKind)
	if err != nil {
		return 0, err
	}

	return realtime.Sequence(seq), nil
}

// watermarkKey は、stream の watermark item のキーを返します。
func watermarkKey(streamID realtime.StreamID) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		attrStreamID: &types.AttributeValueMemberS{Value: string(streamID)},
		attrSequence: &types.AttributeValueMemberN{Value: strconv.Itoa(watermarkSequence)},
	}
}

// ReadAfter は、cursor より後ろを昇順に強い一貫性で読みます。
func (s *store) ReadAfter(ctx context.Context, q realtime.ReadAfterQuery) (realtime.ReadAfterResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	limit := q.Limit
	if limit <= 0 {
		limit = defaultReadLimit
	}

	if limit > maxReadLimit {
		limit = maxReadLimit
	}

	out, err := s.c.Query(ctx, &dynamodb.QueryInput{
		TableName:                aws.String(s.table),
		KeyConditionExpression:   aws.String(attrStreamID + " = :s AND #seq > :after"),
		ExpressionAttributeNames: map[string]string{"#seq": attrSequence},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s":     &types.AttributeValueMemberS{Value: string(q.StreamID)},
			":after": &types.AttributeValueMemberN{Value: q.After.String()},
		},
		ConsistentRead:   aws.Bool(true),
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return realtime.ReadAfterResult{}, dynamodbclient.Normalize(err, "read after")
	}

	events := make([]realtime.DeliveryEvent, 0, len(out.Items))
	for _, item := range out.Items {
		e, err := fromItem(item)
		if err != nil {
			return realtime.ReadAfterResult{}, err
		}

		events = append(events, e)
	}

	return realtime.ReadAfterResult{Events: events, HasMore: len(out.LastEvaluatedKey) > 0}, nil
}

// Latest は、stream の最後の event を降順 1 件の読み取りで返します。
// watermark item は event ではないので、位置で除きます。
func (s *store) Latest(ctx context.Context, streamID realtime.StreamID) (realtime.DeliveryEvent, bool, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	out, err := s.c.Query(ctx, &dynamodb.QueryInput{
		TableName:                aws.String(s.table),
		KeyConditionExpression:   aws.String(attrStreamID + " = :s AND #seq > :watermark"),
		ExpressionAttributeNames: map[string]string{"#seq": attrSequence},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s":         &types.AttributeValueMemberS{Value: string(streamID)},
			":watermark": &types.AttributeValueMemberN{Value: strconv.Itoa(watermarkSequence)},
		},
		ConsistentRead:   aws.Bool(true),
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(1),
	})
	if err != nil {
		return realtime.DeliveryEvent{}, false, dynamodbclient.Normalize(err, "latest")
	}

	if len(out.Items) == 0 {
		return realtime.DeliveryEvent{}, false, nil
	}

	e, err := fromItem(out.Items[0])
	if err != nil {
		return realtime.DeliveryEvent{}, false, err
	}

	return e, true, nil
}

// Find は、指定位置の event を強い一貫性で読みます。
func (s *store) Find(ctx context.Context, streamID realtime.StreamID, seq realtime.Sequence) (realtime.DeliveryEvent, bool, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	out, err := s.c.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(s.table),
		Key:            key(streamID, seq),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return realtime.DeliveryEvent{}, false, dynamodbclient.Normalize(err, "find")
	}

	if len(out.Item) == 0 {
		return realtime.DeliveryEvent{}, false, nil
	}

	e, err := fromItem(out.Item)
	if err != nil {
		return realtime.DeliveryEvent{}, false, err
	}

	return e, true, nil
}

// originAttr は、起点 trace の carrier を item から復元します。無ければ nil です。
func originAttr(item map[string]types.AttributeValue) map[string]string {
	m, ok := item[attrOrigin].(*types.AttributeValueMemberM)
	if !ok || len(m.Value) == 0 {
		return nil
	}

	origin := make(map[string]string, len(m.Value))
	for k, v := range m.Value {
		if s, ok := v.(*types.AttributeValueMemberS); ok {
			origin[k] = s.Value
		}
	}

	return origin
}

// key は、(stream, sequence) の主キーを返します。
func key(streamID realtime.StreamID, seq realtime.Sequence) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		attrStreamID: &types.AttributeValueMemberS{Value: string(streamID)},
		attrSequence: &types.AttributeValueMemberN{Value: seq.String()},
	}
}

// toItem は、封筒を item に写します。expires_at は OccurredAt + EventLogRetention（epoch 秒）で、
// TTL による掃除にだけ使います。
func toItem(e realtime.DeliveryEvent) map[string]types.AttributeValue {
	item := key(e.StreamID, e.Sequence)
	item[attrEventID] = &types.AttributeValueMemberS{Value: e.EventID}
	item[attrType] = &types.AttributeValueMemberS{Value: e.Type}
	item[attrOccurredAt] = &types.AttributeValueMemberS{Value: e.OccurredAt.UTC().Format(time.RFC3339Nano)}
	item[attrSchemaVersion] = &types.AttributeValueMemberN{Value: strconv.Itoa(e.SchemaVersion)}
	item[attrExpiresAt] = &types.AttributeValueMemberN{Value: strconv.FormatInt(e.OccurredAt.Add(realtime.EventLogRetention).Unix(), 10)}
	if len(e.Payload) > 0 {
		item[attrPayload] = &types.AttributeValueMemberB{Value: e.Payload}
	}

	// 起点 trace は無いことがあるので、空の属性を書かずに省きます。
	if len(e.Origin) > 0 {
		origin := make(map[string]types.AttributeValue, len(e.Origin))
		for k, v := range e.Origin {
			origin[k] = &types.AttributeValueMemberS{Value: v}
		}

		item[attrOrigin] = &types.AttributeValueMemberM{Value: origin}
	}

	return item
}

// fromItem は、item を封筒に戻します。形が崩れていれば ErrInternal（store の中身が契約と違う）です。
func fromItem(item map[string]types.AttributeValue) (realtime.DeliveryEvent, error) {
	seq, err := dynamodbclient.NumberAttr(item, attrSequence, itemKind)
	if err != nil {
		return realtime.DeliveryEvent{}, err
	}

	version, err := dynamodbclient.NumberAttr(item, attrSchemaVersion, itemKind)
	if err != nil {
		return realtime.DeliveryEvent{}, err
	}

	if version < 0 || version > math.MaxInt32 {
		return realtime.DeliveryEvent{}, xerrors.Wrap(apperror.ErrInternal, "event log item: schema_version is out of range")
	}

	occurredAt, err := time.Parse(time.RFC3339Nano, dynamodbclient.StringAttr(item, attrOccurredAt))
	if err != nil {
		return realtime.DeliveryEvent{}, xerrors.Wrap(apperror.ErrInternal, "event log item: occurred_at: "+err.Error())
	}

	var payload json.RawMessage
	if b, ok := item[attrPayload].(*types.AttributeValueMemberB); ok {
		payload = json.RawMessage(b.Value)
	}

	return realtime.DeliveryEvent{
		EventID:       dynamodbclient.StringAttr(item, attrEventID),
		StreamID:      realtime.StreamID(dynamodbclient.StringAttr(item, attrStreamID)),
		Sequence:      realtime.Sequence(seq),
		Type:          dynamodbclient.StringAttr(item, attrType),
		OccurredAt:    occurredAt,
		SchemaVersion: int(version),
		Payload:       payload,
		Origin:        originAttr(item),
	}, nil
}
