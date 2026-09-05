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

// FindFeedByUserID は、(ordered_at DESC, id DESC) の keyset で取得し、ステータスと明細の要約を
// 結合で解決します。
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

// FindFeedAll は、所有者の条件を持たない点だけが FindFeedByUserID と異なります。
// 可視範囲の認可は呼び出し側の責務です（docs/spec/usecase/purchase.md 参照）。
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

func toFeedReadModels[T any](rows []T, toRow func(T) feedRow) []query.PurchaseFeedReadModel {
	items := make([]query.PurchaseFeedReadModel, len(rows))
	for i, row := range rows {
		items[i] = toFeedReadModel(toRow(row))
	}
	return items
}

// toFeedReadModel は、合計金額と明細の行数を決済スケール / BIGINT から int へ狭め、
// ステータスと明細の要約には結合先の値を詰めます。
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
