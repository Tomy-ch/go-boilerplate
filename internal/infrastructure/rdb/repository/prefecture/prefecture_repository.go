// Package prefecture は、都道府県リポジトリ（prefecture.Repository）の RDB 実装を提供します。
package prefecture

import (
	"context"

	"go-boilerplate/internal/domain/prefecture"
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

// New は、prefecture.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) prefecture.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindByID は、IDから都道府県エンティティを取得します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*prefecture.Prefecture, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetPrefectureDomainByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToPrefecture(row.ID, row.Name, row.Code)
}

// FindByIDs は、複数IDから都道府県エンティティ一覧を取得します。
func (r *repository) FindByIDs(ctx context.Context, ids []uuid.UUID) (prefecture.Prefectures, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.GetPrefectureDomainByIDs(ctx, ids)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	prefectures := make(prefecture.Prefectures, len(rows))
	for i, row := range rows {
		prefectureEntity, err := rowToPrefecture(row.ID, row.Name, row.Code)
		if err != nil {
			return nil, err
		}
		prefectures[i] = prefectureEntity
	}

	return prefectures, nil
}

// FindByName は、都道府県名から都道府県エンティティを取得します。
func (r *repository) FindByName(ctx context.Context, name string) (*prefecture.Prefecture, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetPrefectureDomainByName(ctx, name)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToPrefecture(row.ID, row.Name, row.Code)
}

// FindAll は、全都道府県エンティティを code 昇順で取得します。
func (r *repository) FindAll(ctx context.Context) (prefecture.Prefectures, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.GetPrefectureDomainAll(ctx)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	prefectures := make(prefecture.Prefectures, len(rows))
	for i, row := range rows {
		prefectureEntity, err := rowToPrefecture(row.ID, row.Name, row.Code)
		if err != nil {
			return nil, err
		}
		prefectures[i] = prefectureEntity
	}

	return prefectures, nil
}

// rowToPrefecture は、DB 行の値からドメインエンティティを再構築します。
func rowToPrefecture(id uuid.UUID, name string, code int16) (*prefecture.Prefecture, error) {
	entity, err := prefecture.New(id, name, int(code))
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}
