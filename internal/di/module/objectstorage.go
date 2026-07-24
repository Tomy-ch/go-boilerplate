package module

import (
	"go-boilerplate/internal/config"                                         // sample-api:line
	s3storage "go-boilerplate/internal/infrastructure/objectstorage/s3"      // sample-api:line
	"go-boilerplate/internal/observability"                                  // sample-api:line
	objectstoragebd "go-boilerplate/internal/usecase/boundary/objectstorage" // sample-api:line

	"go.uber.org/fx"
)

// objectStorageModule は、S3 互換オブジェクトストレージ（boundary.Storage）を提供する fx.Module です。
func objectStorageModule() fx.Option {
	return fx.Module("objectstorage",
		fx.Provide(
			provideObjectStorage, // sample-api:line
		),
	)
}

// provideObjectStorage は、config から S3 アダプタを構築し boundary.Storage を返します。
// 中立境界の背後を S3 実装に隔離し、endpoint / 資格情報の差し替えだけで Garage・MinIO・本番 S3 に接続します。
// sample-api:begin
func provideObjectStorage(cfg *config.ObjectStorageConfig, tf observability.TracerFactory) objectstoragebd.Storage {
	return s3storage.New(s3storage.Config{
		Endpoint:        cfg.Endpoint(),
		Region:          cfg.Region(),
		Bucket:          cfg.Bucket(),
		AccessKeyID:     cfg.AccessKeyID(),
		SecretAccessKey: cfg.SecretAccessKey(),
		UsePathStyle:    cfg.UsePathStyle(),
	}, tf)
}

// sample-api:end
