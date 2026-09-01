// Package inquiry は、問い合わせリポジトリ（inquiry.Repository）の RDB 実装を提供します。
package inquiry

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
)

type repository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、inquiry.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) inquiry.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindByID は、問い合わせを 1 件再構築して返します。存在しない場合は NotFound を返します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*inquiry.Inquiry, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.GetInquiryByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return reconstruct(row.Inquiries)
}

// FindActiveByUserID は、利用者の問い合わせを再構築して返します。
// 存在しない場合は NotFound を返します。
func (r *repository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*inquiry.Inquiry, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.GetInquiryByUserID(ctx, userID)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return reconstruct(row.Inquiries)
}

// Create は、問い合わせを 1 件登録します。
// 同じ利用者の問い合わせが既にある場合は一意制約違反を Conflict として返します。
func (r *repository) Create(ctx context.Context, i *inquiry.Inquiry) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	err := db.CreateInquiry(ctx, &gen.CreateInquiryParams{
		ID:     i.ID(),
		UserID: i.UserID(),
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// Touch は、最後にメッセージが追加された日時を更新します。
func (r *repository) Touch(ctx context.Context, id uuid.UUID, now time.Time) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	err := db.TouchInquiry(ctx, &gen.TouchInquiryParams{ID: id, UpdatedAt: now})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// ListForOperator は、運営向けに問い合わせを更新日時の新しい順で 1 ページ分再構築して返します。
// cursor が nil のときは先頭ページ、それ以外は cursor の続きを返します。
func (r *repository) ListForOperator(
	ctx context.Context,
	params inquiry.ListParams,
) ([]*inquiry.Inquiry, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	pageSize, err := safecast.IntToInt32(params.Limit)
	if err != nil {
		return nil, err
	}

	rows, err := r.listRows(ctx, db, params.Cursor, pageSize)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	inquiries := make([]*inquiry.Inquiry, 0, len(rows))
	for _, row := range rows {
		entity, rerr := reconstruct(row)
		if rerr != nil {
			return nil, rerr
		}
		inquiries = append(inquiries, entity)
	}
	return inquiries, nil
}

// listRows は、cursor の有無で 2 つのクエリを選び、行を同じ型で返します。
func (r *repository) listRows(
	ctx context.Context,
	db *gen.Queries,
	cursor *inquiry.Cursor,
	pageSize int32,
) ([]gen.Inquiries, error) {
	if cursor == nil {
		rows, err := db.ListInquiriesForOperatorFirst(ctx, pageSize)
		if err != nil {
			return nil, err
		}
		return flatten(rows, func(row *gen.ListInquiriesForOperatorFirstRow) gen.Inquiries {
			return row.Inquiries
		}), nil
	}

	rows, err := db.ListInquiriesForOperatorAfter(ctx, &gen.ListInquiriesForOperatorAfterParams{
		CursorUpdatedAt: cursor.UpdatedAt,
		CursorID:        cursor.ID,
		PageSize:        pageSize,
	})
	if err != nil {
		return nil, err
	}
	return flatten(rows, func(row *gen.ListInquiriesForOperatorAfterRow) gen.Inquiries {
		return row.Inquiries
	}), nil
}

// flatten は、埋め込み行の集合から表の行だけを取り出します。
func flatten[T any](rows []*T, pick func(*T) gen.Inquiries) []gen.Inquiries {
	out := make([]gen.Inquiries, 0, len(rows))
	for _, row := range rows {
		out = append(out, pick(row))
	}
	return out
}

// reconstruct は、行をドメイン集約へ写します。
func reconstruct(row gen.Inquiries) (*inquiry.Inquiry, error) {
	return inquiry.Reconstruct(row.ID, inquiry.Attributes{
		UserID:    row.UserID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	})
}
