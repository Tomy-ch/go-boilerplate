// Package purchase は、購入リポジトリ（purchase.Repository）の RDB 実装を提供します。
package purchase

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/kernel/money"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

type repository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、purchase.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) purchase.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindByID は、購入本体と明細の 2 クエリでロックを取らずに読み出し、集約として再構築します
// （ロックを取る LockByID との対）。存在しない場合は NotFound を返します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*purchase.Purchase, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.GetPurchaseByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	detailRows, err := db.ListPurchaseDetailsByPurchaseID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	details, err := toPurchaseDetails(detailRows)
	if err != nil {
		return nil, err
	}

	p := row.Purchases
	entity, err := purchase.Reconstruct(
		p.ID,
		p.Code,
		p.UserID,
		p.StatusID,
		int(row.StatusCode),
		int(p.SubtotalAmount),
		int(p.TaxAmount),
		int(p.ShippingFee),
		int(p.TotalAmount),
		details,
		p.OrderedAt,
		p.PaidAt,
		p.CanceledAt,
		p.ShippedAt,
		p.DeliveredAt,
	)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// LockByID は、購入行のみ悲観ロック（FOR UPDATE OF p）して明細込みで再構築し返します。
// 支払いの状態遷移の競合（同一購入への並行支払い）を購入行ロックで直列化します。存在しない場合は NotFound を返します。
func (r *repository) LockByID(ctx context.Context, id uuid.UUID) (*purchase.Purchase, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.LockPurchaseByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	detailRows, err := db.ListPurchaseDetailsByPurchaseID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	details, err := toPurchaseDetails(detailRows)
	if err != nil {
		return nil, err
	}

	p := row.Purchases
	entity, err := purchase.Reconstruct(
		p.ID,
		p.Code,
		p.UserID,
		p.StatusID,
		int(row.StatusCode),
		int(p.SubtotalAmount),
		int(p.TaxAmount),
		int(p.ShippingFee),
		int(p.TotalAmount),
		details,
		p.OrderedAt,
		p.PaidAt,
		p.CanceledAt,
		p.ShippedAt,
		p.DeliveredAt,
	)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// UpdatePaid は、購入の状態更新（status_id / paid_at）を渡された tx 内で実行します。
// 擬似決済のため単一集約（purchases）のみを更新し、在庫操作は伴いません。
func (r *repository) UpdatePaid(ctx context.Context, p *purchase.Purchase) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	if err := db.UpdatePurchasePaid(ctx, &gen.UpdatePurchasePaidParams{
		StatusCode: toInt16(p.StatusCode()),
		PaidAt:     p.PaidAt(),
		ID:         p.ID(),
	}); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// UpdateShipped は、購入の状態更新（status_id / shipped_at）を渡された tx 内で実行します。
// 配送追跡を扱わないため単一集約（purchases）のみを更新し、在庫操作は伴いません。
// status_id は seed の UUID を焼き込まず purchase_statuses.code から解決します。遷移可否のガードは持たず、
// 対象行が呼び出し側で FOR UPDATE 取得・検証済みであること（ドメインが遷移の source of truth）に依存します。
func (r *repository) UpdateShipped(ctx context.Context, p *purchase.Purchase) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	if err := db.UpdatePurchaseShipped(ctx, &gen.UpdatePurchaseShippedParams{
		StatusCode: toInt16(p.StatusCode()),
		ShippedAt:  p.ShippedAt(),
		ID:         p.ID(),
	}); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// FindDetailByID は、ID から購入詳細（読み取りモデル）を明細込みで取得します。ステータス名は
// 購入ステータスマスタとの結合で解決します。存在しない場合は NotFound を返します。
func (r *repository) FindDetailByID(ctx context.Context, id uuid.UUID) (*purchase.Detail, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.GetPurchaseDetailByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	detailRows, err := db.ListPurchaseDetailsByPurchaseID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	details, err := toPurchaseDetails(detailRows)
	if err != nil {
		return nil, err
	}

	return &purchase.Detail{
		ID:             row.ID,
		Code:           row.Code,
		UserID:         row.UserID,
		StatusID:       row.StatusID,
		StatusName:     row.StatusName,
		SubtotalAmount: int(row.SubtotalAmount),
		TaxAmount:      int(row.TaxAmount),
		ShippingFee:    int(row.ShippingFee),
		TotalAmount:    int(row.TotalAmount),
		Details:        details,
		OrderedAt:      row.OrderedAt,
		PaidAt:         row.PaidAt,
		CanceledAt:     row.CanceledAt,
		ShippedAt:      row.ShippedAt,
	}, nil
}

// toInt16 は、ドメインの int を sqlc の int16（purchase_statuses.code の SMALLINT 列）へ変換します。
func toInt16(v int) int16 {
	//nolint:gosec // G115: 値は purchase_statuses.code（SMALLINT）由来で範囲に収まります
	return int16(v)
}

// toPurchaseDetails は、明細行を購入明細の値オブジェクトへ変換します。単価は価格スケール（ドル decimal）です。
func toPurchaseDetails(detailRows []*gen.ListPurchaseDetailsByPurchaseIDRow) ([]purchase.PurchaseDetail, error) {
	details := make([]purchase.PurchaseDetail, len(detailRows))
	for i, dr := range detailRows {
		d := dr.PurchaseDetails
		unitPrice, perr := money.NewPrice(d.UnitPrice)
		if perr != nil {
			return nil, pgerror.NormalizeReconstructError(perr)
		}
		details[i] = purchase.NewPurchaseDetail(d.ID, d.ProductID, int(d.Quantity), unitPrice)
	}
	return details, nil
}

// FindFeedByUserID は、指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で
// keyset ページネーション取得します。ステータス名は購入ステータスマスタとの結合で解決します。
// params.AfterOrderedAt / AfterID が nil の場合は先頭ページを、それ以外は境界より過去の行を返します。
func (r *repository) FindFeedByUserID(ctx context.Context, userID uuid.UUID, params purchase.ListFeedParams) ([]purchase.FeedItem, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	if params.AfterOrderedAt == nil || params.AfterID == nil {
		rows, err := db.ListPurchasesFeedFirst(ctx, &gen.ListPurchasesFeedFirstParams{
			UserID:     userID,
			LimitParam: params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		items := make([]purchase.FeedItem, len(rows))
		for i, row := range rows {
			items[i] = toFeedItem(row.ID, row.Code, row.TotalAmount, row.OrderedAt, row.StatusID, row.StatusName)
		}
		return items, nil
	}

	rows, err := db.ListPurchasesFeedAfter(ctx, &gen.ListPurchasesFeedAfterParams{
		UserID:         userID,
		AfterOrderedAt: *params.AfterOrderedAt,
		AfterID:        *params.AfterID,
		LimitParam:     params.Limit,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	items := make([]purchase.FeedItem, len(rows))
	for i, row := range rows {
		items[i] = toFeedItem(row.ID, row.Code, row.TotalAmount, row.OrderedAt, row.StatusID, row.StatusName)
	}
	return items, nil
}

// toFeedItem は、購入履歴フィードの行（First / After で別型・同一フィールド）を読み取りモデルへ変換します。
// 合計金額は決済スケール（BIGINT セント）を int へ、ステータス ID / 名称は購入ステータスマスタ由来です。
func toFeedItem(id uuid.UUID, code string, totalAmount int64, orderedAt time.Time, statusID uuid.UUID, statusName string) purchase.FeedItem {
	return purchase.FeedItem{
		Code:        code,
		TotalAmount: int(totalAmount),
		StatusID:    statusID,
		StatusName:  statusName,
		OrderedAt:   orderedAt,
		ID:          id,
	}
}
