// Package dynamodbclient は、DynamoDB 互換 store へ繋ぐ AWS SDK v2 クライアントの組み立てを 1 箇所に置きます。
// Realtime Delivery の 3 つの adapter（eventlog / streamticket / instancelease）と table initializer が
// 同じ接続形と同じ retry 上限で動くための substrate で、boundary interface は持ちません。
package dynamodbclient

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/awsclient"
	"go-boilerplate/pkg/xerrors"
)

const (
	// MaxAttempts は、1 回の API 呼び出しの試行上限（初回 + retry 2 回）です。
	// store が不達のとき無制限に retry せず、呼び出し元へ ErrUnavailable を返して上位の判断
	// （503 + Retry-After、outbox の backoff）に委ねるための固定値です。
	MaxAttempts = 3
	// MaxBackoff は、retry 間の待ち時間の上限です。SSE の write deadline（10 秒）の内側に収めます。
	MaxBackoff = 2 * time.Second
	// tableWaitTimeout は、table が ACTIVE になるのを待つ上限です。
	tableWaitTimeout = 60 * time.Second
)

// Config は、DynamoDB 互換 store への接続設定です。
type Config struct {
	// Endpoint は、DynamoDB 互換エンドポイントです（例 DynamoDB Local: "http://dynamodb_local:8000"）。
	// 空の場合は SDK 既定のエンドポイント解決に委ねます（本番 DynamoDB）。
	Endpoint string
	// Region は、署名に用いるリージョンです。
	Region string
	// AccessKeyID / SecretAccessKey は、明示注入する静的資格情報です。両方空なら
	// SDK 既定の credential chain（IAM ロール等）へ委ねます。詳細は awsclient.Resolve を参照。
	AccessKeyID     string
	SecretAccessKey string
	// HTTPClient は、SDK が API 呼び出しに使う HTTP クライアントです。SSRF ガード付きの実装を DI が
	// 注入します。nil を渡すと SDK 既定のトランスポートになり、ガードを素通りします。
	HTTPClient aws.HTTPClient
}

// TableSpec は、EnsureTable が作る table の定義です。各 adapter package が自分の table の定義を返します。
type TableSpec struct {
	Name                   string
	Attributes             []types.AttributeDefinition
	KeySchema              []types.KeySchemaElement
	GlobalSecondaryIndexes []types.GlobalSecondaryIndex
	// TTLAttribute は、期限切れ item の掃除に使う属性名（epoch 秒）です。空なら TTL を設定しません。
	TTLAttribute string
}

// New は、設定から DynamoDB クライアントを生成します。
// 資格情報を解決できない場合はエラーを返し、認証エラーが最初の操作まで隠れないようにします。
func New(ctx context.Context, cfg Config) (*dynamodb.Client, error) {
	awsCfg, err := awsclient.Resolve(ctx, awsclient.Config{
		Region:          cfg.Region,
		AccessKeyID:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
		HTTPClient:      cfg.HTTPClient,
	})
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		o.RetryMode = aws.RetryModeStandard
		o.RetryMaxAttempts = MaxAttempts
		o.Retryer = retry.NewStandard(func(so *retry.StandardOptions) {
			so.MaxAttempts = MaxAttempts
			so.MaxBackoff = MaxBackoff
		})
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	}), nil
}

// IsConditionalCheckFailed は、条件式が成立せず書き込みが拒否されたエラーかを判定します。
// 冪等な append や cleanup ownership の競合は、この失敗を「他者が先に書いた」として読みます。
func IsConditionalCheckFailed(err error) bool {
	var target *types.ConditionalCheckFailedException

	return xerrors.As(err, &target)
}

// Normalize は、SDK の失敗を apperror sentinel へ正規化します。context の取り消しは ErrCanceled、
// それ以外は ErrUnavailable です。条件式の不成立は呼び出し側が IsConditionalCheckFailed で先に判定します。
func Normalize(err error, op string) error {
	if xerrors.Is(err, context.Canceled) || xerrors.Is(err, context.DeadlineExceeded) {
		return xerrors.Wrap(apperror.ErrCanceled, op+": "+err.Error())
	}

	return xerrors.Wrap(apperror.ErrUnavailable, op+": "+err.Error())
}

// EnsureTable は、table が無ければ作り、ACTIVE を待ち、TTL が未設定なら設定します。
// 既にある table に対しては何も変えずに成功するので、何度実行しても同じ状態に収束します
// （application の起動時ではなく one-shot の初期化で呼びます）。
func EnsureTable(ctx context.Context, c *dynamodb.Client, spec TableSpec) error {
	_, err := c.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:              aws.String(spec.Name),
		AttributeDefinitions:   spec.Attributes,
		KeySchema:              spec.KeySchema,
		GlobalSecondaryIndexes: spec.GlobalSecondaryIndexes,
		BillingMode:            types.BillingModePayPerRequest,
	})
	if err != nil && !isResourceInUse(err) {
		return Normalize(err, "create table "+spec.Name)
	}

	waiter := dynamodb.NewTableExistsWaiter(c)
	if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(spec.Name)}, tableWaitTimeout); err != nil {
		return Normalize(err, "wait table "+spec.Name)
	}

	if spec.TTLAttribute == "" {
		return nil
	}

	return ensureTimeToLive(ctx, c, spec.Name, spec.TTLAttribute)
}

// ensureTimeToLive は、TTL が attr で有効（または有効化中）でなければ設定します。
func ensureTimeToLive(ctx context.Context, c *dynamodb.Client, table, attr string) error {
	desc, err := c.DescribeTimeToLive(ctx, &dynamodb.DescribeTimeToLiveInput{TableName: aws.String(table)})
	if err != nil {
		return Normalize(err, "describe ttl "+table)
	}

	if d := desc.TimeToLiveDescription; d != nil && aws.ToString(d.AttributeName) == attr &&
		(d.TimeToLiveStatus == types.TimeToLiveStatusEnabled || d.TimeToLiveStatus == types.TimeToLiveStatusEnabling) {
		return nil
	}

	_, err = c.UpdateTimeToLive(ctx, &dynamodb.UpdateTimeToLiveInput{
		TableName:               aws.String(table),
		TimeToLiveSpecification: &types.TimeToLiveSpecification{AttributeName: aws.String(attr), Enabled: aws.Bool(true)},
	})
	if err != nil {
		return Normalize(err, "update ttl "+table)
	}

	return nil
}

// isResourceInUse は、table が既にあるために CreateTable が拒否されたかを判定します。
func isResourceInUse(err error) bool {
	var target *types.ResourceInUseException

	return xerrors.As(err, &target)
}

// StringAttr は、item の S 属性を返します。無い、または S でなければ空文字です。
func StringAttr(item map[string]types.AttributeValue, name string) string {
	if s, ok := item[name].(*types.AttributeValueMemberS); ok {
		return s.Value
	}

	return ""
}

// NumberAttr は、item の N 属性を int64 として返します。無い・N でない・整数に読めない場合は
// ErrInternal です（この adapter が書いた覚えの無い形が store にあることを意味します）。
// kind はエラー文に載せる item の種類（例 "event log"）です。
func NumberAttr(item map[string]types.AttributeValue, name, kind string) (int64, error) {
	n, ok := item[name].(*types.AttributeValueMemberN)
	if !ok {
		return 0, xerrors.Wrap(apperror.ErrInternal, kind+" item: "+name+" is not a number")
	}

	v, err := strconv.ParseInt(n.Value, 10, 64)
	if err != nil {
		return 0, xerrors.Wrap(apperror.ErrInternal, kind+" item: "+name+": "+err.Error())
	}

	return v, nil
}
