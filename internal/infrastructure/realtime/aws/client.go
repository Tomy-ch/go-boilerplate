package aws

import (
	"context"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	smithymiddleware "github.com/aws/smithy-go/middleware"

	"go-boilerplate/internal/infrastructure/awsclient"
)

// ClientConfig は、SNS / SQS クライアントの接続設定です。1 つの endpoint が両方の API を受けます
// （GoAWS も本番も同じ組み立てで、差は endpoint と資格情報だけ）。
type ClientConfig struct {
	// Endpoint は、SNS / SQS 互換エンドポイントです。空なら SDK 既定の解決（本番 AWS）に委ねます。
	Endpoint string
	// Region は、署名に用いるリージョンです。
	Region string
	// AccessKeyID / SecretAccessKey は、明示注入する静的資格情報です。両方空なら SDK 既定の
	// credential chain へ委ねます。詳細は awsclient.Resolve を参照。
	AccessKeyID     string
	SecretAccessKey string
	// HTTPClient は、SDK が API 呼び出しに使う HTTP クライアントです。nil を渡すと SDK 既定の
	// トランスポートになり、SSRF ガードを素通りします。
	HTTPClient awssdk.HTTPClient
}

// Clients は、同じ資格情報で組み立てた SNS / SQS クライアントの組です。
type Clients struct {
	SNS *sns.Client
	SQS *sqs.Client
}

// NewClients は、設定から SNS / SQS クライアントを生成します。資格情報の解決は 1 回で、失敗すれば
// エラーを返し、認証エラーが最初の publish / receive まで隠れないようにします。
const (
	// CallTimeout は、1 回の API 呼び出し全体（retry を含む）に与える上限です。
	//
	// これが無いと、応答を返さない substrate に対して呼び出しがいつまでも戻りません。SDK にも HTTP
	// クライアントにも別の上限がなく（DI が渡すクライアントは Timeout を持たず、SSRF ガードのために
	// 差し替えた Dialer にも Timeout はありません）、awsclient.Resolve の上限は起動時の資格情報解決
	// 1 回にしか掛かりません。one-shot の job は既定で全体のタイムアウトを持たないため、1 呼び出しが
	// 戻らないとジョブごと戻らなくなります。
	//
	// 値は dynamodbclient.CallTimeout に揃えます。同じ機構が使う 2 つの substrate で、1 呼び出しに
	// 与える猶予を変える理由がないためです。
	CallTimeout = 10 * time.Second
	// receiveCallTimeout は、long polling する受信にだけ与える上限です。受信は receiveWaitSeconds
	// （20 秒）まで意図的に待つので、CallTimeout を当てると毎回 deadline で落ちて loop が回りません。
	// 待ち時間そのものではなく「待ち切ってなお戻らない」ことを捕まえる値にします。
	receiveCallTimeout = (receiveWaitSeconds + 10) * time.Second
	// receiveOperation は、long polling する操作の名前です（smithy が middleware へ渡す値）。
	receiveOperation = "ReceiveMessage"
)

func NewClients(ctx context.Context, cfg ClientConfig) (Clients, error) {
	awsCfg, err := awsclient.Resolve(ctx, awsclient.Config{
		Region:          cfg.Region,
		AccessKeyID:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
		HTTPClient:      cfg.HTTPClient,
	})
	if err != nil {
		return Clients{}, err
	}

	return Clients{
		SNS: sns.NewFromConfig(awsCfg, func(o *sns.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = awssdk.String(cfg.Endpoint)
			}
			o.APIOptions = append(o.APIOptions, withCallTimeout())
		}),
		SQS: sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = awssdk.String(cfg.Endpoint)
			}
			o.APIOptions = append(o.APIOptions, withCallTimeout())
		}),
	}, nil
}

// withCallTimeout は、API 呼び出し 1 回の全体に上限を与える middleware を stack の先頭へ差します。
// retry の外側に置くので、上限は試行ごとではなく呼び出し全体に掛かります。long polling する受信だけは
// 待ち時間が仕様なので別の上限を当てます。
func withCallTimeout() func(*smithymiddleware.Stack) error {
	return func(stack *smithymiddleware.Stack) error {
		return stack.Initialize.Add(
			smithymiddleware.InitializeMiddlewareFunc(
				"realtimeCallTimeout",
				func(
					ctx context.Context, in smithymiddleware.InitializeInput, next smithymiddleware.InitializeHandler,
				) (smithymiddleware.InitializeOutput, smithymiddleware.Metadata, error) {
					d := CallTimeout
					if awsmiddleware.GetOperationName(ctx) == receiveOperation {
						d = receiveCallTimeout
					}

					ctx, cancel := context.WithTimeout(ctx, d)
					defer cancel()

					return next.HandleInitialize(ctx, in)
				},
			),
			smithymiddleware.Before,
		)
	}
}
