package module

import (
	"go-boilerplate/internal/config"                       // sample-api:line
	domainproduct "go-boilerplate/internal/domain/product" // sample-api:line
	"go-boilerplate/internal/observability"
	addressuc "go-boilerplate/internal/usecase/address"                      // sample-api:line
	authzbd "go-boilerplate/internal/usecase/boundary/authz"                 // sample-api:line
	objectstoragebd "go-boilerplate/internal/usecase/boundary/objectstorage" // sample-api:line
	exchangerateuc "go-boilerplate/internal/usecase/exchangerate"            // sample-api:line
	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/outbox"
	prefectureuc "go-boilerplate/internal/usecase/prefecture"     // sample-api:line
	productuc "go-boilerplate/internal/usecase/product"           // sample-api:line
	categoryuc "go-boilerplate/internal/usecase/product/category" // sample-api:line
	statusuc "go-boilerplate/internal/usecase/product/status"     // sample-api:line
	purchaseuc "go-boilerplate/internal/usecase/purchase"         // sample-api:line
	"go-boilerplate/internal/usecase/user"                        // sample-api:line
	"go-boilerplate/internal/usecase/user/search"                 // sample-api:line

	"go.uber.org/fx"
)

// UsecaseModule は、ユースケース層の依存関係を提供するfx.Moduleです。
func UsecaseModule() fx.Option {
	return fx.Module("usecase",
		fx.Provide(
			healthcheck.New,
			// 具象 IdempotencyMetrics を usecase 境界の Metrics / GCMetrics の双方として供給する。
			fx.Annotate(
				observability.NewIdempotencyMetrics,
				fx.As(new(idempotency.Metrics)),
				fx.As(new(idempotency.GCMetrics)),
			),
			idempotency.NewDeps,
			idempotency.NewGC,
			outbox.NewEmit,
			outbox.NewGC,
			outbox.NewReplay,
			// sample-api:begin
			// サンプルのユースケース
			user.New,
			search.New,
			exchangerateuc.New,
			addressuc.New,
			prefectureuc.New,
			statusuc.New,
			categoryuc.New,
			provideProductUsecase,
			purchaseuc.New,
			// sample-api:end
		),
	)
}

// provideProductUsecase は、商品ユースケースを object storage / authz / アップロード上限（config 由来）とともに構築します。
// sample-api:begin
func provideProductUsecase(
	repo domainproduct.Repository,
	storage objectstoragebd.Storage,
	authorizer authzbd.Authorizer,
	cfg *config.ObjectStorageConfig,
	tf observability.TracerFactory,
) productuc.Usecase {
	return productuc.New(repo, storage, authorizer, cfg.MaxUploadBytes(), tf)
}

// sample-api:end
