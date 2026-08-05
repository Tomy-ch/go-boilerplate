// Package objectstorage は、オブジェクトストレージ境界（boundary.Storage）の実装を選ぶ唯一の場所です。
// 背後の substrate を差し替える場合に書き換えるのはこのパッケージだけで、DI も CLI もここを通ります。
package objectstorage

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/objectstorage/s3"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/objectstorage"
)

// New は、config から S3 互換アダプタを構築し boundary.Storage を返します。
// endpoint / 資格情報の差し替えだけで Garage・MinIO・本番 S3 のいずれにも接続します。
func New(
	ctx context.Context,
	cfg *config.ObjectStorageConfig,
	outbound *observability.OutboundHTTPClient,
	tf observability.TracerFactory,
) (boundary.Storage, error) {
	return s3.New(ctx, s3.Config{
		Endpoint:        cfg.Endpoint(),
		Region:          cfg.Region(),
		Bucket:          cfg.Bucket(),
		AccessKeyID:     cfg.AccessKeyID(),
		SecretAccessKey: cfg.SecretAccessKey(),
		UsePathStyle:    cfg.UsePathStyle(),
		HTTPClient:      outbound,
	}, tf)
}
