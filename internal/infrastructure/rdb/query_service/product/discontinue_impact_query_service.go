package product

import (
	"context"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/query"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
)

type discontinueImpactService struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// NewDiscontinueImpactQueryService は、廃番の影響見積もりクエリサービスの RDB 実装を生成して返します。
func NewDiscontinueImpactQueryService(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.DiscontinueImpactQueryService {
	return &discontinueImpactService{
		db:     db,
		tracer: tf.Infra(),
	}
}

// EstimateDiscontinueImpact は、商品を廃番にした場合の影響を件数で取得します。
// 3 つの件数は母集団が異なるため独立した問い合わせで数えます。実行側との対応関係は
// [query.DiscontinueImpactQueryService.EstimateDiscontinueImpact] を参照。
func (s *discontinueImpactService) EstimateDiscontinueImpact(
	ctx context.Context,
	productID uuid.UUID,
) (query.DiscontinueImpactReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))

	carts, err := db.CountDiscontinueImpactCarts(ctx, productID)
	if err != nil {
		return query.DiscontinueImpactReadModel{}, pgerror.NormalizeError(err)
	}

	users, err := db.CountDiscontinueImpactUsers(ctx, productID)
	if err != nil {
		return query.DiscontinueImpactReadModel{}, pgerror.NormalizeError(err)
	}

	// 「進行中」の定義は購入集約が持つため、終端の code をドメインから受け取って渡します。
	terminal := purchase.TerminalStatusCodes()
	codes := make([]int16, len(terminal))
	for i, c := range terminal {
		converted, cerr := safecast.IntToInt16(c)
		if cerr != nil {
			return query.DiscontinueImpactReadModel{}, cerr
		}
		codes[i] = converted
	}

	purchases, err := db.CountDiscontinueImpactInProgressPurchases(ctx, &gen.CountDiscontinueImpactInProgressPurchasesParams{
		ProductID:           productID,
		TerminalStatusCodes: codes,
	})
	if err != nil {
		return query.DiscontinueImpactReadModel{}, pgerror.NormalizeError(err)
	}

	return query.DiscontinueImpactReadModel{
		AffectedCartCount:       carts,
		AffectedUserCount:       users,
		InProgressPurchaseCount: purchases,
	}, nil
}
