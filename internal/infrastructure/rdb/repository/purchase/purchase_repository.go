// Package purchase は、購入リポジトリ（purchase.Repository）の RDB 実装を提供します。
package purchase

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
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
	entity, err := purchase.Reconstruct(p.ID, purchase.Attributes{
		Code:           p.Code,
		UserID:         p.UserID,
		StatusID:       p.StatusID,
		StatusCode:     int(row.StatusCode),
		SubtotalAmount: int(p.SubtotalAmount),
		TaxAmount:      int(p.TaxAmount),
		ShippingFee:    int(p.ShippingFee),
		TotalAmount:    int(p.TotalAmount),
		Details:        details,
		OrderedAt:      p.OrderedAt,
		PaidAt:         p.PaidAt,
		CanceledAt:     p.CanceledAt,
		ShippedAt:      p.ShippedAt,
		DeliveredAt:    p.DeliveredAt,
	})
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
	entity, err := purchase.Reconstruct(p.ID, purchase.Attributes{
		Code:           p.Code,
		UserID:         p.UserID,
		StatusID:       p.StatusID,
		StatusCode:     int(row.StatusCode),
		SubtotalAmount: int(p.SubtotalAmount),
		TaxAmount:      int(p.TaxAmount),
		ShippingFee:    int(p.ShippingFee),
		TotalAmount:    int(p.TotalAmount),
		Details:        details,
		OrderedAt:      p.OrderedAt,
		PaidAt:         p.PaidAt,
		CanceledAt:     p.CanceledAt,
		ShippedAt:      p.ShippedAt,
		DeliveredAt:    p.DeliveredAt,
	})
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// UpdatePaid は、購入の状態更新（status_id / paid_at）を渡された tx 内で実行します。
func (r *repository) UpdatePaid(ctx context.Context, p *purchase.Purchase) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	statusCode, err := safecast.IntToInt16(p.StatusCode())
	if err != nil {
		return xerrors.Wrap(err, "invalid purchase status code")
	}

	db := gen.New(driver.New(ctx, r.db))
	if err := db.UpdatePurchasePaid(ctx, &gen.UpdatePurchasePaidParams{
		StatusCode: statusCode,
		PaidAt:     p.PaidAt(),
		ID:         p.ID(),
	}); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// UpdateShipped は、購入の状態更新（status_id / shipped_at）を渡された tx 内で実行します。
// status_id は seed の UUID を焼き込まず purchase_statuses.code から解決します。遷移可否のガードは持たず、
// 対象行が呼び出し側で FOR UPDATE 取得・検証済みであること（ドメインが遷移の source of truth）に依存します。
func (r *repository) UpdateShipped(ctx context.Context, p *purchase.Purchase) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	statusCode, err := safecast.IntToInt16(p.StatusCode())
	if err != nil {
		return xerrors.Wrap(err, "invalid purchase status code")
	}

	db := gen.New(driver.New(ctx, r.db))
	if err := db.UpdatePurchaseShipped(ctx, &gen.UpdatePurchaseShippedParams{
		StatusCode: statusCode,
		ShippedAt:  p.ShippedAt(),
		ID:         p.ID(),
	}); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// UpdateDelivered は、購入の状態更新（status_id / delivered_at）を渡された tx 内で実行します。
// status_id は seed の UUID を焼き込まず purchase_statuses.code から解決します。遷移可否のガードは持たず、
// 対象行が呼び出し側で FOR UPDATE 取得・検証済みであること（ドメインが遷移の source of truth）に依存します。
func (r *repository) UpdateDelivered(ctx context.Context, p *purchase.Purchase) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	statusCode, err := safecast.IntToInt16(p.StatusCode())
	if err != nil {
		return xerrors.Wrap(err, "invalid purchase status code")
	}

	db := gen.New(driver.New(ctx, r.db))
	if err := db.UpdatePurchaseDelivered(ctx, &gen.UpdatePurchaseDeliveredParams{
		StatusCode:  statusCode,
		DeliveredAt: p.DeliveredAt(),
		ID:          p.ID(),
	}); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// FindShippable は、発送可能な購入を注文日時の古い順（同時刻は ID 昇順）で最大 limit 件、
// 明細込みで再構築して返します。
//
// status_id は seed の UUID を焼き込まず purchase_statuses.code で絞り込みます。
func (r *repository) FindShippable(ctx context.Context, limit int32) (purchase.Purchases, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	statusCode, err := safecast.IntToInt16(purchase.StatusPaid.Code())
	if err != nil {
		return nil, xerrors.Wrap(err, "invalid purchase status code")
	}

	db := gen.New(driver.New(ctx, r.db))

	rows, err := db.ListShippablePurchases(ctx, &gen.ListShippablePurchasesParams{
		StatusCode: statusCode,
		LimitParam: limit,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	if len(rows) == 0 {
		return purchase.Purchases{}, nil
	}

	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.Purchases.ID
	}

	detailRows, err := db.ListPurchaseDetailsByPurchaseIDs(ctx, ids)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	detailsByPurchaseID, err := groupPurchaseDetails(detailRows)
	if err != nil {
		return nil, err
	}

	purchases := make(purchase.Purchases, len(rows))
	for i, row := range rows {
		p := row.Purchases
		entity, rerr := purchase.Reconstruct(p.ID, purchase.Attributes{
			Code:           p.Code,
			UserID:         p.UserID,
			StatusID:       p.StatusID,
			StatusCode:     int(row.StatusCode),
			SubtotalAmount: int(p.SubtotalAmount),
			TaxAmount:      int(p.TaxAmount),
			ShippingFee:    int(p.ShippingFee),
			TotalAmount:    int(p.TotalAmount),
			Details:        detailsByPurchaseID[p.ID],
			OrderedAt:      p.OrderedAt,
			PaidAt:         p.PaidAt,
			CanceledAt:     p.CanceledAt,
			ShippedAt:      p.ShippedAt,
			DeliveredAt:    p.DeliveredAt,
		})
		if rerr != nil {
			return nil, pgerror.NormalizeReconstructError(rerr)
		}
		purchases[i] = entity
	}
	return purchases, nil
}

// groupPurchaseDetails は、複数購入分の明細行を購入 ID ごとにまとめます。
// 各購入内の並びは取得順（明細 ID 昇順）を保ちます。
func groupPurchaseDetails(
	detailRows []*gen.ListPurchaseDetailsByPurchaseIDsRow,
) (map[uuid.UUID][]purchase.PurchaseDetail, error) {
	grouped := make(map[uuid.UUID][]purchase.PurchaseDetail)
	for _, dr := range detailRows {
		d := dr.PurchaseDetails
		detail, err := toPurchaseDetail(d)
		if err != nil {
			return nil, err
		}
		grouped[d.PurchaseID] = append(grouped[d.PurchaseID], detail)
	}
	return grouped, nil
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
		DeliveredAt:    row.DeliveredAt,
	}, nil
}

// toPurchaseDetail は、明細行 1 件を購入明細の値オブジェクトへ変換します。単価は価格スケール（ドル decimal）です。
func toPurchaseDetail(d gen.PurchaseDetails) (purchase.PurchaseDetail, error) {
	unitPrice, err := money.NewPrice(d.UnitPrice)
	if err != nil {
		return purchase.PurchaseDetail{}, pgerror.NormalizeReconstructError(err)
	}
	return purchase.NewPurchaseDetail(d.ID, purchase.PurchaseDetailAttributes{
		ProductID: d.ProductID,
		Quantity:  int(d.Quantity),
		UnitPrice: unitPrice,
	}), nil
}

// toPurchaseDetails は、購入 1 件分の明細行を購入明細の値オブジェクトへ変換します。
func toPurchaseDetails(detailRows []*gen.ListPurchaseDetailsByPurchaseIDRow) ([]purchase.PurchaseDetail, error) {
	details := make([]purchase.PurchaseDetail, len(detailRows))
	for i, dr := range detailRows {
		detail, err := toPurchaseDetail(dr.PurchaseDetails)
		if err != nil {
			return nil, err
		}
		details[i] = detail
	}
	return details, nil
}

// FindFeedByUserID は、指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で
// keyset ページネーション取得します。ステータス名は購入ステータスマスタとの結合で解決します。
// params.AfterOrderedAt / AfterID が nil の場合は先頭ページを、それ以外は境界より過去の行を返します。
// params.OrderedAfter / OrderedBefore が揃っている場合は、その半開区間に注文された購入だけを返します。
func (r *repository) FindFeedByUserID(ctx context.Context, userID uuid.UUID, params purchase.ListFeedParams) ([]purchase.FeedItem, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	filterByPeriod := params.OrderedAfter != nil && params.OrderedBefore != nil

	if params.AfterOrderedAt == nil || params.AfterID == nil {
		rows, err := db.ListPurchasesFeedFirst(ctx, &gen.ListPurchasesFeedFirstParams{
			UserID:         userID,
			FilterByPeriod: filterByPeriod,
			OrderedAfter:   params.OrderedAfter,
			OrderedBefore:  params.OrderedBefore,
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

	rows, err := db.ListPurchasesFeedAfter(ctx, &gen.ListPurchasesFeedAfterParams{
		UserID:         userID,
		AfterOrderedAt: *params.AfterOrderedAt,
		AfterID:        *params.AfterID,
		FilterByPeriod: filterByPeriod,
		OrderedAfter:   params.OrderedAfter,
		OrderedBefore:  params.OrderedBefore,
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

// FindStatusesByUserID は、指定ユーザーの購入が取っているステータスを重複なく取得します。
// 進行中かどうかでは絞り込まず、code は purchase_statuses との結合で解決してドメインの値へ復元します。
func (r *repository) FindStatusesByUserID(ctx context.Context, userID uuid.UUID) ([]purchase.Status, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	codes, err := db.SelectPurchaseStatusCodesByUserID(ctx, userID)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	statuses := make([]purchase.Status, len(codes))
	for i, code := range codes {
		status, serr := purchase.NewStatus(int(code))
		if serr != nil {
			return nil, serr
		}
		statuses[i] = status
	}
	return statuses, nil
}

// FindUserIDsWithPurchases は、purchases に 1 件以上の行を持つ user_id を重複排除して返します。
// ステータスでは絞り込まず、users とは結合しません（集約をまたぐ結合を避けるため ID 群の照会に切り出しています）。
func (r *repository) FindUserIDsWithPurchases(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	ids, err := db.ListUserIDsWithPurchases(ctx, userIDs)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return ids, nil
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
