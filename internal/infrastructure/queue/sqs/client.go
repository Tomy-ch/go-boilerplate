package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"go-boilerplate/internal/infrastructure/awsclient"
)

// ClientConfig は、SQS クライアントの接続設定です。
// endpoint / 資格情報の差し替えだけで ElasticMQ・LocalStack・本番 SQS のいずれにも接続できます。
type ClientConfig struct {
	// Endpoint は、SQS 互換エンドポイントです（例 ElasticMQ: "http://elasticmq:9324"）。
	// 空の場合は SDK 既定のエンドポイント解決に委ねます（本番 AWS SQS 等）。
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

// NewClient は、設定から SQS クライアントを生成します。
// 資格情報を解決できない場合はエラーを返し、認証エラーが最初の送受信まで隠れないようにします。
func NewClient(ctx context.Context, cfg ClientConfig) (*sqs.Client, error) {
	awsCfg, err := awsclient.Resolve(ctx, awsclient.Config{
		Region:          cfg.Region,
		AccessKeyID:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
		HTTPClient:      cfg.HTTPClient,
	})
	if err != nil {
		return nil, err
	}

	return sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	}), nil
}
