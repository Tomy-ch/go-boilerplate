package module

import (
	"go-boilerplate/internal/config"                                 // sample-api:line
	domainproduct "go-boilerplate/internal/domain/product"           // sample-api:line
	domaincategory "go-boilerplate/internal/domain/product/category" // sample-api:line
	domainstatus "go-boilerplate/internal/domain/product/status"     // sample-api:line
	"go-boilerplate/internal/observability"
	addressuc "go-boilerplate/internal/usecase/address"                      // sample-api:line
	authzbd "go-boilerplate/internal/usecase/boundary/authz"                 // sample-api:line
	objectstoragebd "go-boilerplate/internal/usecase/boundary/objectstorage" // sample-api:line
	txbd "go-boilerplate/internal/usecase/boundary/tx"                       // sample-api:line
	dashboarduc "go-boilerplate/internal/usecase/dashboard"                  // sample-api:line
	exchangerateuc "go-boilerplate/internal/usecase/exchangerate"            // sample-api:line
	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/outbox"
	prefectureuc "go-boilerplate/internal/usecase/prefecture"            // sample-api:line
	productuc "go-boilerplate/internal/usecase/product"                  // sample-api:line
	categoryuc "go-boilerplate/internal/usecase/product/category"        // sample-api:line
	rankinguc "go-boilerplate/internal/usecase/product/ranking"          // sample-api:line
	statusuc "go-boilerplate/internal/usecase/product/status"            // sample-api:line
	purchaseuc "go-boilerplate/internal/usecase/purchase"                // sample-api:line
	purchasesummaryuc "go-boilerplate/internal/usecase/purchase/summary" // sample-api:line
	"go-boilerplate/internal/usecase/user"                               // sample-api:line
	"go-boilerplate/internal/usecase/user/search"                        // sample-api:line

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
			user.NewPurge,
			search.New,
			exchangerateuc.New,
			addressuc.New,
			prefectureuc.New,
			statusuc.New,
			categoryuc.New,
			rankinguc.New,
			provideProductUsecase,
			purchaseuc.New,
			purchasesummaryuc.New,
			dashboarduc.New,
			// sample-api:end
		),
	)
}

// sample-api:begin
// provideProductUsecase は、商品ユースケースを tx / マスタ Repository / object storage / authz /
// アップロード上限（config 由来）とともに構築します。
func provideProductUsecase(
	txm txbd.Manager,
	repo domainproduct.Repository,
	categoryRepo domaincategory.Repository,
	statusRepo domainstatus.Repository,
	storage objectstoragebd.Storage,
	authorizer authzbd.Authorizer,
	cfg *config.ObjectStorageConfig,
	tf observability.TracerFactory,
) productuc.Usecase {
	return productuc.New(txm, repo, categoryRepo, statusRepo, storage, authorizer, cfg.MaxUploadBytes(), tf)
}

// sample-api:end
