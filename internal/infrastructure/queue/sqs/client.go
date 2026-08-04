package sqs

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// ClientConfig は、SQS クライアントの接続設定です。
// endpoint / 資格情報の差し替えだけで ElasticMQ・LocalStack・本番 SQS のいずれにも接続できます。
// 資格情報は静的注入のみを扱い、SDK 既定の credential chain（IAM ロール等）へは委譲しません。
// ロール運用へ移す場合は本関数の差し替えが要ります。
type ClientConfig struct {
	// Endpoint は、SQS 互換エンドポイントです（例 ElasticMQ: "http://elasticmq:9324"）。
	// 空の場合は SDK 既定のエンドポイント解決に委ねます（本番 AWS SQS 等）。
	Endpoint string
	// Region は、署名に用いるリージョンです。
	Region string
	// AccessKeyID は、静的資格情報のアクセスキー ID です。
	AccessKeyID string
	// SecretAccessKey は、静的資格情報のシークレットアクセスキーです。
	SecretAccessKey string
	// HTTPClient は、SDK が使う HTTP クライアントです。SSRF ガード付きの実装を DI が注入します。
	// nil を渡すと SDK 既定のトランスポートになり、ガードを素通りします。
	HTTPClient aws.HTTPClient
}

// NewClient は、設定から SQS クライアントを生成します。
func NewClient(cfg ClientConfig) *sqs.Client {
	awsCfg := aws.Config{
		Region:      cfg.Region,
		Credentials: awscreds.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		HTTPClient:  cfg.HTTPClient,
	}
	return sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
}
