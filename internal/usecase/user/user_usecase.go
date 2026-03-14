//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package user は、ユーザーに関するユースケースを提供します。
package user

import (
	"context"

	"boilerplate-go/internal/domain/prefecture"
	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/boundary/clock"
	"boilerplate-go/internal/usecase/boundary/security"
	"boilerplate-go/internal/usecase/boundary/tx"
	"boilerplate-go/internal/usecase/tools/paging"
	"boilerplate-go/internal/usecase/tools/search"
	"boilerplate-go/internal/usecase/user/query"
	"boilerplate-go/pkg/uuid"
)

// MutableFields は、ユーザー取得結果のDTOを表します。
type MutableFields struct {
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	PostalCode     string
	PrefectureName string
	City           string
	Street         string
	Building       *string
}

// GetParamsDTO は、ユーザー取得に必要なパラメータを表します。
type GetParamsDTO struct {
	Keyword *string
	Active  *bool
}

// CreateParamsDTO は、ユーザー作成に必要なパラメータを表します。
type CreateParamsDTO struct {
	UserID      uuid.UUID
	RawPassword string

	MutableFields
}

// usecase は、ユーザーに関するユースケースを提供します。
type usecase struct {
	tracer      observability.LayerTracer
	txm         tx.Manager
	clock       clock.Clock
	byencrypter security.Bcrypter
	userRepo    user.Repository
	pftRepo     prefecture.Repository
	userQS      query.UserQueryService
}

// Usecase は、ユーザーに関するユースケースを定義します。
type Usecase interface {
	// ListUsersByKeyword は、ユーザー一覧を取得します。
	ListUsersByKeyword(ctx context.Context, params *GetParamsDTO, page *paging.Paging) ([]MutableFields, error)
	// CreateUser は、ユーザーを作成します。
	CreateUser(ctx context.Context, dto *CreateParamsDTO) (MutableFields, error)
	// CountUsers は、ユーザーの総件数を返します。
	CountUsers(ctx context.Context, active *bool) (int64, error)
}

// New は、ユーザーに関するユースケースを初期化します。
func New(
	tf observability.TracerFactory,
	txm tx.Manager,
	clock clock.Clock,
	byencrypter security.Bcrypter,
	userRepo user.Repository,
	prefectureRepo prefecture.Repository,
	userQueryService query.UserQueryService,
) Usecase {
	return &usecase{
		tracer:      tf.Usecase(),
		txm:         txm,
		clock:       clock,
		byencrypter: byencrypter,
		userRepo:    userRepo,
		pftRepo:     prefectureRepo,
		userQS:      userQueryService,
	}
}

// ListUsersByKeyword は、ユーザー一覧を取得するユースケースです。
func (u *usecase) ListUsersByKeyword(ctx context.Context, params *GetParamsDTO, page *paging.Paging) ([]MutableFields, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	var (
		us  user.Users
		err error
	)

	if params != nil {
		keywords := search.ParseSearchTokens(params.Keyword, search.DefaultMaxTokens)
		us, err = u.userQS.FindByKeyword(ctx, keywords, params.Active, page.Limit32(), page.Offset32())
	} else {
		us, err = u.userRepo.FindAll(ctx, page.Limit32(), page.Offset32())
	}

	if err != nil {
		return nil, err
	}

	ctx, prefectureMap, err := observability.RunDomainWithSpan(
		ctx, u.tracer, "user", "prefectureMap", func(ctx context.Context) (map[uuid.UUID]*prefecture.Entity, error) {
			pids := make([]uuid.UUID, len(us))
			for i, u := range us {
				pids[i] = u.PrefectureID()
			}

			ps, pftErr := u.pftRepo.FindByIDs(ctx, pids)
			if pftErr != nil {
				return nil, pftErr
			}

			prefectureMap := make(map[uuid.UUID]*prefecture.Entity, len(ps))
			for _, p := range ps {
				prefectureMap[p.ID()] = p
			}

			return prefectureMap, nil
		})
	if err != nil {
		return nil, err
	}

	_, dtos, err := observability.RunDomainWithSpan(
		ctx, u.tracer, "user", "buildDTOs", func(_ context.Context) ([]MutableFields, error) {
			dtos := make([]MutableFields, len(us))
			for i, u := range us {
				dtos[i] = MutableFields{
					FirstName:  u.FirstName(),
					LastName:   u.LastName(),
					Email:      u.Email(),
					Phone:      u.Phone(),
					PostalCode: u.PostalCode(),
					City:       u.City(),
					Street:     u.Street(),
					Building:   u.Building(),
				}
				if p, ok := prefectureMap[us[i].PrefectureID()]; ok {
					dtos[i].PrefectureName = p.Name()
				}
			}
			return dtos, nil
		})

	return dtos, err
}

// CreateUser は、ユーザーを作成するユースケースです。
func (u *usecase) CreateUser(ctx context.Context, dto *CreateParamsDTO) (MutableFields, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()
	passwordHash, err := u.byencrypter.Hash(dto.RawPassword)
	if err != nil {
		return MutableFields{}, err
	}

	var (
		userEntity *user.User
		pftDomain  *prefecture.Entity
	)
	err = u.txm.Do(ctx, func(ctx context.Context) error {
		var err error
		pftDomain, err = u.pftRepo.FindByName(ctx, dto.PrefectureName)
		if err != nil {
			return err
		}

		userEntity, err = user.New(
			dto.UserID,
			dto.FirstName,
			dto.LastName,
			passwordHash,
			dto.Email,
			dto.Phone,
			pftDomain.ID(),
			dto.City,
			dto.Street,
			dto.Building,
			dto.PostalCode,
			now,
			now,
			nil,
		)
		if err != nil {
			return err
		}

		err = u.userRepo.Create(ctx, userEntity)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return MutableFields{}, err
	}

	return MutableFields{
		FirstName:      userEntity.FirstName(),
		LastName:       userEntity.LastName(),
		Email:          userEntity.Email(),
		Phone:          userEntity.Phone(),
		PostalCode:     userEntity.PostalCode(),
		PrefectureName: pftDomain.Name(),
		City:           userEntity.City(),
		Street:         userEntity.Street(),
		Building:       userEntity.Building(),
	}, nil
}

// CountUsers は、ユーザーの総件数を返すユースケースです。
func (u *usecase) CountUsers(ctx context.Context, active *bool) (int64, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()
	return u.userRepo.CountByActive(ctx, active)
}
