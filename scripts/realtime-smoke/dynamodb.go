package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"go-boilerplate/pkg/xerrors"
)

const (
	ddbSubject = "DynamoDB Local"

	attrStream   = "stream_id"
	attrSequence = "sequence"
	attrPayload  = "payload"
	attrExpires  = "expires_at"

	// streamID は、検査で使う唯一の partition key です。順序の検査は 1 stream の中で行います。
	streamID = "smoke"
	// seedCount は、順序 / pagination 検査のために append する item 数です。
	seedCount = 5
	// cursorAfter は、`sequence > cursor` 読みの cursor です。seedCount より小さい値なら何でもよいです。
	cursorAfter = 2
	// pageLimit は、pagination 検査の 1 ページ件数です。seedCount を割り切らない値にして最終ページを短くします。
	pageLimit = 2

	tableWaitTimeout = 30 * time.Second

	codeConditionalCheckFailed = "ConditionalCheckFailedException"
)

// ddbSmoke は、DynamoDB 検査の状態です。
type ddbSmoke struct {
	c     *dynamodb.Client
	table string
}

// runDynamoDB は、EventLog が依存する呼び出しを順に検査し、結果を記録します。
func runDynamoDB(ctx context.Context, c *dynamodb.Client, table string, keep bool, rec *recorder) {
	s := &ddbSmoke{c: c, table: table}

	created := runChain(ctx, ddbSubject, s.steps(), rec)
	s.cleanup(ctx, created, keep, rec)
}

func (s *ddbSmoke) steps() []step {
	return []step{
		{id: "D1", check: "CreateTable + TableExists waiter", halt: true, fn: s.createTable},
		{id: "D2", check: "conditional PutItem（attribute_not_exists）", halt: true, fn: s.conditionalPut},
		{id: "D3", check: "同一 key の再 PutItem → ConditionalCheckFailedException", fn: s.duplicatePut},
		{id: "D4", check: "Query ConsistentRead=true, sequence > cursor（昇順・欠番なし）", fn: s.queryAfterCursor},
		{id: "D5", check: "Query ScanIndexForward=false Limit=1（最新 1 件）", fn: s.queryLatest},
		{id: "D6", check: "Query Limit + ExclusiveStartKey の pagination", fn: s.queryPaginated},
		{id: "D7", check: "UpdateTimeToLive + DescribeTimeToLive", fn: s.timeToLive},
	}
}

func (s *ddbSmoke) createTable(ctx context.Context) (string, error) {
	_, err := s.c.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(s.table),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String(attrStream), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(attrSequence), AttributeType: types.ScalarAttributeTypeN},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(attrStream), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String(attrSequence), KeyType: types.KeyTypeRange},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		return "", err
	}

	waiter := dynamodb.NewTableExistsWaiter(s.c)
	if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(s.table)}, tableWaitTimeout); err != nil {
		return "", xerrors.Wrap(err, "wait table exists")
	}

	return "table " + s.table, nil
}

// putSequence は、stream 内の 1 sequence を conditional put します。EventLog の append と同じ形です。
func (s *ddbSmoke) putSequence(ctx context.Context, seq int) error {
	_, err := s.c.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item: map[string]types.AttributeValue{
			attrStream:   &types.AttributeValueMemberS{Value: streamID},
			attrSequence: &types.AttributeValueMemberN{Value: strconv.Itoa(seq)},
			attrPayload:  &types.AttributeValueMemberS{Value: "payload-" + strconv.Itoa(seq)},
		},
		ConditionExpression: aws.String("attribute_not_exists(" + attrStream + ")"),
	})

	return err
}

func (s *ddbSmoke) conditionalPut(ctx context.Context) (string, error) {
	if err := s.putSequence(ctx, 1); err != nil {
		return "", err
	}

	return "sequence 1 を append", nil
}

func (s *ddbSmoke) duplicatePut(ctx context.Context) (string, error) {
	err := s.putSequence(ctx, 1)
	if err == nil {
		return "", incompatible("条件式が無視され、同一 key の PutItem が成功した")
	}

	var api apiError
	if xerrors.As(err, &api) && api.ErrorCode() == codeConditionalCheckFailed {
		return "型付きの " + codeConditionalCheckFailed + " が返った", nil
	}

	return "", err
}

func (s *ddbSmoke) seed(ctx context.Context) error {
	for seq := 2; seq <= seedCount; seq++ {
		if err := s.putSequence(ctx, seq); err != nil {
			return xerrors.Wrap(err, "seed sequence "+strconv.Itoa(seq))
		}
	}

	return nil
}

func (s *ddbSmoke) query(ctx context.Context, in *dynamodb.QueryInput) ([]int, map[string]types.AttributeValue, error) {
	in.TableName = aws.String(s.table)
	// 式で参照しない名前を渡すと production の DynamoDB も ValidationException を返すため、使う式にだけ付ける。
	if strings.Contains(aws.ToString(in.KeyConditionExpression), "#seq") {
		in.ExpressionAttributeNames = map[string]string{"#seq": attrSequence}
	}

	out, err := s.c.Query(ctx, in)
	if err != nil {
		return nil, nil, err
	}

	seqs := make([]int, 0, len(out.Items))
	for _, item := range out.Items {
		n, ok := item[attrSequence].(*types.AttributeValueMemberN)
		if !ok {
			return nil, nil, incompatible("sequence が N 型で返らない")
		}

		v, err := strconv.Atoi(n.Value)
		if err != nil {
			return nil, nil, incompatible("sequence が整数で返らない: " + n.Value)
		}

		seqs = append(seqs, v)
	}

	return seqs, out.LastEvaluatedKey, nil
}

func (s *ddbSmoke) queryAfterCursor(ctx context.Context) (string, error) {
	if err := s.seed(ctx); err != nil {
		return "", err
	}

	seqs, _, err := s.query(ctx, &dynamodb.QueryInput{
		KeyConditionExpression: aws.String(attrStream + " = :s AND #seq > :c"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s": &types.AttributeValueMemberS{Value: streamID},
			":c": &types.AttributeValueMemberN{Value: strconv.Itoa(cursorAfter)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}

	want := make([]int, 0, seedCount-cursorAfter)
	for seq := cursorAfter + 1; seq <= seedCount; seq++ {
		want = append(want, seq)
	}

	if !equalInts(seqs, want) {
		return "", incompatible("期待 " + joinInts(want) + " に対し " + joinInts(seqs) + " が返った")
	}

	return joinInts(seqs) + " が昇順で返った", nil
}

func (s *ddbSmoke) queryLatest(ctx context.Context) (string, error) {
	seqs, _, err := s.query(ctx, &dynamodb.QueryInput{
		KeyConditionExpression:    aws.String(attrStream + " = :s"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":s": &types.AttributeValueMemberS{Value: streamID}},
		ScanIndexForward:          aws.Bool(false),
		Limit:                     aws.Int32(1),
		ConsistentRead:            aws.Bool(true),
	})
	if err != nil {
		return "", err
	}

	if !equalInts(seqs, []int{seedCount}) {
		return "", incompatible("最新 1 件として " + joinInts(seqs) + " が返った")
	}

	return "sequence " + strconv.Itoa(seedCount) + " が返った", nil
}

func (s *ddbSmoke) queryPaginated(ctx context.Context) (string, error) {
	var (
		got   []int
		pages int
		start map[string]types.AttributeValue
	)

	for {
		seqs, last, err := s.query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression:    aws.String(attrStream + " = :s"),
			ExpressionAttributeValues: map[string]types.AttributeValue{":s": &types.AttributeValueMemberS{Value: streamID}},
			Limit:                     aws.Int32(pageLimit),
			ExclusiveStartKey:         start,
			ConsistentRead:            aws.Bool(true),
		})
		if err != nil {
			return "", err
		}

		pages++
		got = append(got, seqs...)

		if last == nil || pages > seedCount {
			break
		}

		start = last
	}

	want := make([]int, 0, seedCount)
	for seq := 1; seq <= seedCount; seq++ {
		want = append(want, seq)
	}

	if !equalInts(got, want) {
		return "", incompatible("pagination の結果が " + joinInts(got) + "（" + strconv.Itoa(pages) + " ページ）")
	}

	return strconv.Itoa(pages) + " ページで " + strconv.Itoa(len(got)) + " 件（LastEvaluatedKey で継続）", nil
}

func (s *ddbSmoke) timeToLive(ctx context.Context) (string, error) {
	_, err := s.c.UpdateTimeToLive(ctx, &dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String(s.table),
		TimeToLiveSpecification: &types.TimeToLiveSpecification{
			AttributeName: aws.String(attrExpires),
			Enabled:       aws.Bool(true),
		},
	})
	if err != nil {
		return "", err
	}

	out, err := s.c.DescribeTimeToLive(ctx, &dynamodb.DescribeTimeToLiveInput{TableName: aws.String(s.table)})
	if err != nil {
		return "", err
	}

	desc := out.TimeToLiveDescription
	if desc == nil || aws.ToString(desc.AttributeName) != attrExpires {
		return "", incompatible("UpdateTimeToLive は受理されたが DescribeTimeToLive に反映されない")
	}

	switch desc.TimeToLiveStatus {
	case types.TimeToLiveStatusEnabled, types.TimeToLiveStatusEnabling:
		return string(desc.TimeToLiveStatus) + " が読み戻った（DynamoDB Local は期限切れ item を削除しない — 既知の caveat）", nil
	case types.TimeToLiveStatusDisabled, types.TimeToLiveStatusDisabling:
		return "", incompatible("TTL の状態が " + string(desc.TimeToLiveStatus))
	default:
		return "", incompatible("TTL の状態が不明: " + string(desc.TimeToLiveStatus))
	}
}

// cleanup は、作成した table を削除します。-keep のときは行として残し、黙って省きません。
func (s *ddbSmoke) cleanup(ctx context.Context, created, keep bool, rec *recorder) {
	const id, check = "D8", "DeleteTable（後片付け）"

	switch {
	case !created:
		rec.skip(id, ddbSubject, check, "先行検査 D1 により実行不能")
	case keep:
		rec.skip(id, ddbSubject, check, "-keep により未実施（table "+s.table+" が残っている）")
	default:
		_, err := s.c.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(s.table)})
		rec.record(id, ddbSubject, check, "table "+s.table+" を削除", err)
	}
}

func equalInts(a, b []int) bool {
	return slices.Equal(a, b)
}

func joinInts(v []int) string {
	return fmt.Sprint(v)
}
