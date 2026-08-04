// Package s3 は、オブジェクトストレージ境界（objectstorage.Storage）の S3 互換実装を提供します。
// AWS SDK v2 S3 クライアントを用い、endpoint / 資格情報の差し替えだけで Garage・MinIO・本番 S3 の
// いずれにも接続できます。
package s3

import (
	"bytes"
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

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

	input := &awss3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(obj.Key),
		Body:          bytes.NewReader(obj.Body),
		ContentType:   aws.String(obj.ContentType),
		ContentLength: aws.Int64(int64(len(obj.Body))),
	}
	// 空の Cache-Control ヘッダを送るとキャッシュ指示の無い状態と区別できないため、未指定のままにする。
	if obj.CacheControl != "" {
		input.CacheControl = aws.String(obj.CacheControl)
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", xerrors.Wrap(apperror.ErrUnavailable, "objectstorage put failed: "+err.Error())
	}
	return boundary.Path(obj.Key), nil
}

// List は、バケットから query.Prefix 配下のオブジェクトを 1 ページ分列挙します。
// 続きがある場合は NextCursor に S3 の continuation token を載せて返します。
// S3 の失敗は apperror.ErrUnavailable へ正規化します。
func (s *storage) List(ctx context.Context, query boundary.ListQuery) (boundary.ListResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	input := &awss3.ListObjectsV2Input{Bucket: aws.String(s.bucket)}
	// 空の Prefix / ContinuationToken を送ると、S3 互換実装によっては全件の先頭からと解釈されない。
	// 未指定はフィールドを立てないことで表す。
	if query.Prefix != "" {
		input.Prefix = aws.String(query.Prefix)
	}
	if query.Cursor != "" {
		input.ContinuationToken = aws.String(query.Cursor)
	}
	if query.Limit > 0 {
		input.MaxKeys = aws.Int32(query.Limit)
	}

	out, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return boundary.ListResult{}, xerrors.Wrap(apperror.ErrUnavailable, "objectstorage list failed: "+err.Error())
	}

	objects := make([]boundary.Object, 0, len(out.Contents))
	for _, c := range out.Contents {
		objects = append(objects, boundary.Object{
			Key:        aws.ToString(c.Key),
			ModifiedAt: aws.ToTime(c.LastModified),
		})
	}
	result := boundary.ListResult{Objects: objects}
	// IsTruncated が false のときの token は続きを表さないため、境界には最終ページを表す空文字だけを渡す。
	if aws.ToBool(out.IsTruncated) {
		result.NextCursor = aws.ToString(out.NextContinuationToken)
	}
	return result, nil
}

// Delete は、keys のオブジェクトをバケットからまとめて削除します。
// S3 の DeleteObjects は存在しないキーも成功として扱うため、再実行しても結果は変わりません。
// S3 の失敗は apperror.ErrUnavailable へ正規化します。
func (s *storage) Delete(ctx context.Context, keys []string) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	if len(keys) == 0 {
		return nil
	}

	ids := make([]s3types.ObjectIdentifier, 0, len(keys))
	for _, k := range keys {
		ids = append(ids, s3types.ObjectIdentifier{Key: aws.String(k)})
	}

	out, err := s.client.DeleteObjects(ctx, &awss3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &s3types.Delete{Objects: ids, Quiet: aws.Bool(true)},
	})
	if err != nil {
		return xerrors.Wrap(apperror.ErrUnavailable, "objectstorage delete failed: "+err.Error())
	}
	// DeleteObjects は一部のキーだけが失敗しても呼び出し自体は成功を返す。
	// 消えていないキーを消えたものとして扱うと GC が同じ孤児を報告しなくなるため、明示的に失敗させる。
	if len(out.Errors) > 0 {
		return xerrors.Wrap(apperror.ErrUnavailable,
			"objectstorage delete failed for "+strconv.Itoa(len(out.Errors))+" keys: "+aws.ToString(out.Errors[0].Message))
	}
	return nil
}
