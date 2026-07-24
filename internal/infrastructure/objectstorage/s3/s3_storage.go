// Package s3 は、オブジェクトストレージ境界（objectstorage.Storage）の S3 互換実装を提供します。
// AWS SDK v2 S3 クライアントを用い、endpoint / 資格情報の差し替えだけで Garage・MinIO・本番 S3 の
// いずれにも接続できます。
package s3

import (
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/objectstorage"
	"go-boilerplate/pkg/xerrors"
)

// Config は、S3 互換オブジェクトストレージへの接続設定です（DI で注入）。
type Config struct {
	// Endpoint は、S3 互換エンドポイントです（例 Garage: "http://garage:3900"）。
	// 空の場合は SDK 既定のエンドポイント解決に委ねます（本番 AWS S3 等）。
	Endpoint string
	// Region は、署名に用いるリージョンです。
	Region string
	// Bucket は、オブジェクトを格納するバケット名です。
	Bucket string
	// AccessKeyID は、静的資格情報のアクセスキー ID です。
	AccessKeyID string
	// SecretAccessKey は、静的資格情報のシークレットアクセスキーです。
	SecretAccessKey string
	// UsePathStyle は、path-style アクセスを使うかどうかです（Garage / MinIO は true が必要）。
	UsePathStyle bool
}

// storage は、boundary.Storage の S3 互換実装です。
type storage struct {
	client *awss3.Client
	bucket string
	tracer observability.LayerTracer
}

// New は、S3 互換オブジェクトストレージ実装を生成します。
func New(cfg Config, tf observability.TracerFactory) boundary.Storage {
	awsCfg := aws.Config{
		Region:      cfg.Region,
		Credentials: awscreds.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
	}
	client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return &storage{
		client: client,
		bucket: cfg.Bucket,
		tracer: tf.Infra(),
	}
}

// Put は、obj をバケットの指定キーへ保存し、保存されたパス（キー）を返します。
// S3 の失敗は apperror.ErrUnavailable へ正規化します。
func (s *storage) Put(ctx context.Context, obj boundary.PutObject) (boundary.Path, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	_, err := s.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(obj.Key),
		Body:          bytes.NewReader(obj.Body),
		ContentType:   aws.String(obj.ContentType),
		ContentLength: aws.Int64(int64(len(obj.Body))),
	})
	if err != nil {
		return "", xerrors.Wrap(apperror.ErrUnavailable, "objectstorage put failed: "+err.Error())
	}
	return boundary.Path(obj.Key), nil
}
