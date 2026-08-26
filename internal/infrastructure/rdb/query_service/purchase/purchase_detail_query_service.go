// Package purchase は、購入詳細クエリサービス（query.PurchaseDetailQueryService）の RDB 実装を提供します。
package purchase

import (
	"context"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/pkg/uuid"
)

type service struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、購入詳細クエリサービスの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.PurchaseDetailQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindDetailByUserAndCode は、認証主体（userID）が所有する購入 1 件を購入コードで引き、
// 明細（商品名込み）とともに取得します。
// 所有権は本体クエリの WHERE 述語（user_id 一致）で担保し、他人の購入・不存在はいずれも 0 行 →
// NotFound で秘匿します。明細は products との結合で商品名を解決する固定 2 クエリ構成で N+1 を避けます。
func (s *service) FindDetailByUserAndCode(ctx context.Context, userID uuid.UUID, code string) (*query.PurchaseDetailReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))

	row, err := db.GetPurchaseDetailForUser(ctx, &gen.GetPurchaseDetailForUserParams{
		Code:   code,
		UserID: userID,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	itemRows, err := db.ListPurchaseDetailItemsForUser(ctx, row.ID)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	items, err := toPurchaseDetailItems(itemRows)
	if err != nil {
		return nil, err
	}

	return &query.PurchaseDetailReadModel{
		ID:             row.ID,
		Code:           row.Code,
		UserID:         row.UserID,
		StatusID:       row.StatusID,
		StatusCode:     int(row.StatusCode),
		StatusName:     row.StatusName,
		SubtotalAmount: row.SubtotalAmount,
		TaxAmount:      row.TaxAmount,
		ShippingFee:    row.ShippingFee,
		TotalAmount:    row.TotalAmount,
		Items:          items,
		OrderedAt:      row.OrderedAt,
		PaidAt:         row.PaidAt,
		CanceledAt:     row.CanceledAt,
	}, nil
}

// toPurchaseDetailItems は、明細行を読み取りモデルへ変換します。単価は価格スケール（ドル decimal）で、
// 値オブジェクト再構築に失敗した行は内部エラーへ正規化します。
func toPurchaseDetailItems(rows []*gen.ListPurchaseDetailItemsForUserRow) ([]query.PurchaseDetailItem, error) {
	items := make([]query.PurchaseDetailItem, len(rows))
	for i, r := range rows {
		unitPrice, perr := money.NewPrice(r.UnitPrice)
		if perr != nil {
			return nil, pgerror.NormalizeReconstructError(perr)
		}
		items[i] = query.PurchaseDetailItem{
			ProductID:   r.ProductID,
			ProductName: r.ProductName,
			Quantity:    int(r.Quantity),
			UnitPrice:   unitPrice,
		}
	}
	return items, nil
}
