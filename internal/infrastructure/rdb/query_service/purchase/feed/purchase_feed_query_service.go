// Package feed は、購入履歴一覧クエリサービス（query.PurchaseFeedQueryService）の RDB 実装を提供します。
package feed

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/pkg/uuid"
)

// feedRow は、購入履歴フィードの行を表す中間表現です。sqlc は先頭ページと keyset 境界のクエリごとに
// 別の行型を生成しますが列は同一のため、変換を 1 つに揃えるためにここで受け直します。
type feedRow struct {
	ID            uuid.UUID
	Code          string
	TotalAmount   int64
	OrderedAt     time.Time
	StatusID      uuid.UUID
	StatusCode    int16
	StatusName    string
	FirstItemName string
	ItemCount     int64
}

type service struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、購入履歴一覧クエリサービスの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.PurchaseFeedQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindFeedByUserID は、指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で
// keyset ページネーション取得します。ステータスは購入ステータスマスタ、明細の要約は商品との結合で
// 解決します。params.AfterOrderedAt / AfterID が nil の場合は先頭ページを、それ以外は境界より過去の行を返します。
// params.Window が境界を持つ場合は、その半開区間に注文された購入だけを返します。
func (s *service) FindFeedByUserID(
	ctx context.Context, userID uuid.UUID, params query.ListFeedParams,
) ([]query.PurchaseFeedReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	if params.AfterOrderedAt == nil || params.AfterID == nil {
		rows, err := db.ListPurchasesFeedFirst(ctx, &gen.ListPurchasesFeedFirstParams{
			UserID:        userID,
			OrderedAfter:  params.Window.After(),
			OrderedBefore: params.Window.Before(),
			StatusCodes:   params.StatusCodes,
			ProductID:     params.ProductID,
			LimitParam:    params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return toFeedReadModels(rows, func(row *gen.ListPurchasesFeedFirstRow) feedRow {
			return feedRow{
				ID: row.ID, Code: row.Code, TotalAmount: row.TotalAmount, OrderedAt: row.OrderedAt,
				StatusID: row.StatusID, StatusCode: row.StatusCode, StatusName: row.StatusName,
				FirstItemName: row.FirstItemName, ItemCount: row.ItemCount,
			}
		}), nil
	}

	rows, err := db.ListPurchasesFeedAfter(ctx, &gen.ListPurchasesFeedAfterParams{
		UserID:         userID,
		AfterOrderedAt: *params.AfterOrderedAt,
		AfterID:        *params.AfterID,
		OrderedAfter:   params.Window.After(),
		OrderedBefore:  params.Window.Before(),
		StatusCodes:    params.StatusCodes,
		ProductID:      params.ProductID,
		LimitParam:     params.Limit,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return toFeedReadModels(rows, func(row *gen.ListPurchasesFeedAfterRow) feedRow {
		return feedRow{
			ID: row.ID, Code: row.Code, TotalAmount: row.TotalAmount, OrderedAt: row.OrderedAt,
			StatusID: row.StatusID, StatusCode: row.StatusCode, StatusName: row.StatusName,
			FirstItemName: row.FirstItemName, ItemCount: row.ItemCount,
		}
	}), nil
}

// FindFeedAll は、購入者を問わず購入履歴を FindFeedByUserID と同じ順序・同じ絞り込みで取得します。
// 所有権で閉じないため、可視範囲の認可は呼び出し側の責務です（docs/spec/usecase/purchase.md 参照）。
func (s *service) FindFeedAll(
	ctx context.Context, params query.ListFeedParams,
) ([]query.PurchaseFeedReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	if params.AfterOrderedAt == nil || params.AfterID == nil {
		rows, err := db.ListAllPurchasesFeedFirst(ctx, &gen.ListAllPurchasesFeedFirstParams{
			OrderedAfter:  params.Window.After(),
			OrderedBefore: params.Window.Before(),
			StatusCodes:   params.StatusCodes,
			ProductID:     params.ProductID,
			LimitParam:    params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return toFeedReadModels(rows, func(row *gen.ListAllPurchasesFeedFirstRow) feedRow {
			return feedRow{
				ID: row.ID, Code: row.Code, TotalAmount: row.TotalAmount, OrderedAt: row.OrderedAt,
				StatusID: row.StatusID, StatusCode: row.StatusCode, StatusName: row.StatusName,
				FirstItemName: row.FirstItemName, ItemCount: row.ItemCount,
			}
		}), nil
	}

	rows, err := db.ListAllPurchasesFeedAfter(ctx, &gen.ListAllPurchasesFeedAfterParams{
		AfterOrderedAt: *params.AfterOrderedAt,
		AfterID:        *params.AfterID,
		OrderedAfter:   params.Window.After(),
		OrderedBefore:  params.Window.Before(),
		StatusCodes:    params.StatusCodes,
		ProductID:      params.ProductID,
		LimitParam:     params.Limit,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return toFeedReadModels(rows, func(row *gen.ListAllPurchasesFeedAfterRow) feedRow {
		return feedRow{
			ID: row.ID, Code: row.Code, TotalAmount: row.TotalAmount, OrderedAt: row.OrderedAt,
			StatusID: row.StatusID, StatusCode: row.StatusCode, StatusName: row.StatusName,
			FirstItemName: row.FirstItemName, ItemCount: row.ItemCount,
		}
	}), nil
}

// toFeedReadModels は、呼び出し側が渡す変換関数で各行を feedRow へ揃えてから読み取りモデルの列へ写像します。
func toFeedReadModels[T any](rows []T, toRow func(T) feedRow) []query.PurchaseFeedReadModel {
	items := make([]query.PurchaseFeedReadModel, len(rows))
	for i, row := range rows {
		items[i] = toFeedReadModel(toRow(row))
	}
	return items
}

// toFeedReadModel は、購入履歴フィードの行を読み取りモデルへ変換します。
// 合計金額と明細の行数は決済スケール / BIGINT を int へ、ステータスと明細の要約は結合先の値です。
func toFeedReadModel(row feedRow) query.PurchaseFeedReadModel {
	return query.PurchaseFeedReadModel{
		Code:          row.Code,
		TotalAmount:   int(row.TotalAmount),
		StatusID:      row.StatusID,
		StatusCode:    int(row.StatusCode),
		StatusName:    row.StatusName,
		FirstItemName: row.FirstItemName,
		ItemCount:     int(row.ItemCount),
		OrderedAt:     row.OrderedAt,
		ID:            row.ID,
	}
}
