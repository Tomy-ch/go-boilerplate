package module

import (
	"context"

	"go-boilerplate/internal/config"
	objectstorage "go-boilerplate/internal/infrastructure/objectstorage"
	"go-boilerplate/internal/observability"
	objectstoragebd "go-boilerplate/internal/usecase/boundary/objectstorage"

	"go.uber.org/fx"
)

// objectStorageModule は、S3 互換オブジェクトストレージ（boundary.Storage）を提供する fx.Module です。
func objectStorageModule() fx.Option {
	return fx.Module("objectstorage",
		fx.Provide(
			provideObjectStorage,
		),
	)
}

// provideObjectStorage は、オブジェクトストレージ実装を組み立てて boundary.Storage を返します。
// 資格情報を解決できない場合はエラーを返し、app.Start を失敗させます。
func provideObjectStorage(
	cfg *config.ObjectStorageConfig,
	outbound *observability.OutboundHTTPClient,
	tf observability.TracerFactory,
) (objectstoragebd.Storage, error) {
	return objectstorage.New(context.Background(), cfg, outbound, tf)
}
