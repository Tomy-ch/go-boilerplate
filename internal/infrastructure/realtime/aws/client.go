package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

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
	// HTTPClient は、SDK が API 呼び出しに使う HTTP クライアントです。SSRF ガード付きの実装を DI が
	// 注入します。nil を渡すと SDK 既定のトランスポートになり、ガードを素通りします。
	HTTPClient awssdk.HTTPClient
}

// Clients は、同じ資格情報で組み立てた SNS / SQS クライアントの組です。
type Clients struct {
	SNS *sns.Client
	SQS *sqs.Client
}

// NewClients は、設定から SNS / SQS クライアントを生成します。資格情報の解決は 1 回で、失敗すれば
// エラーを返し、認証エラーが最初の publish / receive まで隠れないようにします。
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
		}),
		SQS: sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = awssdk.String(cfg.Endpoint)
			}
		}),
	}, nil
}
