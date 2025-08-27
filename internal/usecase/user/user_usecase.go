//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock$GOPACKAGE

// Package useruc は、ユーザーに関するユースケースを提供します。
package useruc

import (
	"context"

	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/usecase/paging"
)

// DTO は、ユーザーに関するデータ転送用のオブジェクトです。
type DTO struct {
	Name  string
	Email string
	Phone string
}

// usecase は、ユーザーに関するユースケースを提供します。
type usecase struct {
	userRepo user.Repository
}

// Usecase は、ユーザーに関するユースケースを定義します。
type Usecase interface {
	GetAllUsers(ctx context.Context, page paging.Paging) ([]DTO, error)
}

// New は、ユーザーに関するユースケースを初期化します。
func New(userRepo user.Repository) Usecase {
	return &usecase{
		userRepo: userRepo,
	}
}

// GetAllUsers は、ユーザー一覧を取得するユースケースです。
func (u *usecase) GetAllUsers(ctx context.Context, page paging.Paging) ([]DTO, error) {
	us, err := u.userRepo.GetAllUsers(ctx, page.Limit(), page.Offset())
	if err != nil {
		return nil, err
	}
	dtos := make([]DTO, len(us))
	for i, u := range us {
		dtos[i] = DTO{
			Name:  u.FullName(),
			Phone: u.Phone(),
			Email: u.Email(),
		}
	}
	return dtos, nil
}
