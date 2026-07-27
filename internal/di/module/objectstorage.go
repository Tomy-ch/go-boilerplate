package module

import (
	"go-boilerplate/internal/config"
	s3storage "go-boilerplate/internal/infrastructure/objectstorage/s3"
	"go-boilerplate/internal/observability"
	objectstoragebd "go-boilerplate/internal/usecase/boundary/objectstorage"

	"go.uber.org/fx"
)

// objectStorageModule は、S3 互換オブジェクトストレージ（boundary.Storage）を提供する fx.Module です。
func objectStorageModule() fx.Option {
	return fx.Module("objectstorage",
		fx.Provide(
			NewObjectStorage,
		),
	)
}

// NewObjectStorage は、config から S3 アダプタを構築し boundary.Storage を返します。
// 中立境界の背後を S3 実装に隔離し、endpoint / 資格情報の差し替えだけで Garage・MinIO・本番 S3 に接続します。
//
// 実装の選択をここ 1 箇所に閉じ込めるため、DI グラフを組まない CLI もこの関数を通します。
// S3 互換でない substrate へ移す場合に書き換えるのはこの関数だけです。
func NewObjectStorage(cfg *config.ObjectStorageConfig, tf observability.TracerFactory) objectstoragebd.Storage {
	return s3storage.New(s3storage.Config{
		Endpoint:        cfg.Endpoint(),
		Region:          cfg.Region(),
		Bucket:          cfg.Bucket(),
		AccessKeyID:     cfg.AccessKeyID(),
		SecretAccessKey: cfg.SecretAccessKey(),
		UsePathStyle:    cfg.UsePathStyle(),
	}, tf)
}
