//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package prefecture は、都道府県マスタの参照ユースケースを提供します。
package prefecture

import (
	"context"

	"go-boilerplate/internal/domain/prefecture"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

// PrefectureDTO は、都道府県 1 件分のユースケース出力 DTO です。
type PrefectureDTO struct {
	ID   uuid.UUID
	Code int
	Name string
}

// PrefectureDTOs は、PrefectureDTO の一覧（code 昇順）です。
type PrefectureDTOs []PrefectureDTO

// Usecase は、都道府県マスタの参照ユースケースを定義します。
type Usecase interface {
	// ListPrefectures は、全都道府県を code 昇順で返します。
	ListPrefectures(ctx context.Context) (PrefectureDTOs, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	repo   prefecture.Repository
}

// New は、都道府県マスタの参照ユースケースを生成します。
func New(repo prefecture.Repository, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		repo:   repo,
	}
}

// ListPrefectures は、全都道府県を code 昇順で返します。
func (u *usecase) ListPrefectures(ctx context.Context) (PrefectureDTOs, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	prefectures, err := u.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make(PrefectureDTOs, len(prefectures))
	for i, p := range prefectures {
		dtos[i] = PrefectureDTO{
			ID:   p.ID(),
			Code: p.Code(),
			Name: p.Name(),
		}
	}

	return dtos, nil
}
