//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package user は、ユーザーに関するユースケースを提供します。
package user

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/prefecture"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/security"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
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
	DeletedAt      *time.Time
}

// CreateParamsDTO は、ユーザー作成に必要なパラメータを表します。
type CreateParamsDTO struct {
	UserID      uuid.UUID
	RawPassword string

	MutableFields
}

// usecase は、ユーザーに関するユースケースを提供します。
type usecase struct {
	tracer    observability.LayerTracer
	txm       tx.Manager
	clock     clock.Clock
	encrypter security.Encrypter
	userRepo  user.Repository
	pftRepo   prefecture.Repository
}

// Usecase は、ユーザーに関するユースケースを定義します。
type Usecase interface {
	// ListUsers は、ユーザー一覧を取得します。
	ListUsers(ctx context.Context, active *bool, page *paging.Paging) ([]MutableFields, error)
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
	encrypter security.Encrypter,
	userRepo user.Repository,
	prefectureRepo prefecture.Repository,
) Usecase {
	return &usecase{
		tracer:    tf.Usecase(),
		txm:       txm,
		clock:     clock,
		encrypter: encrypter,
		userRepo:  userRepo,
		pftRepo:   prefectureRepo,
	}
}

func (u *usecase) ListUsers(ctx context.Context, active *bool, page *paging.Paging) ([]MutableFields, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	us, err := u.userRepo.FindByActive(ctx, active, page.Limit32(), page.Offset32())
	if err != nil {
		return nil, err
	}

	_, prefectureMap, err := observability.RunWithSpan(
		ctx, u.tracer, "usecase", "user", "prefectureMap", func(ctx context.Context) (map[uuid.UUID]*prefecture.Prefecture, error) {
			pids := make([]uuid.UUID, len(us))
			for i, u := range us {
				pids[i] = u.PrefectureID()
			}

			ps, pftErr := u.pftRepo.FindByIDs(ctx, pids)
			if pftErr != nil {
				return nil, pftErr
			}

			prefectureMap := make(map[uuid.UUID]*prefecture.Prefecture, len(ps))
			for _, p := range ps {
				prefectureMap[p.ID()] = p
			}

			return prefectureMap, nil
		})
	if err != nil {
		return nil, err
	}

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
			DeletedAt:  u.DeletedAt(),
		}
		if p, ok := prefectureMap[us[i].PrefectureID()]; ok {
			dtos[i].PrefectureName = p.Name()
		}
	}

	return dtos, nil
}

// CreateUser は、ユーザーを作成するユースケースです。
func (u *usecase) CreateUser(ctx context.Context, dto *CreateParamsDTO) (MutableFields, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()
	rawPassword, err := user.NewRawPassword(dto.RawPassword)
	if err != nil {
		return MutableFields{}, err
	}

	passwordHash, err := u.encrypter.Hash(rawPassword.Value())
	if err != nil {
		return MutableFields{}, err
	}

	var (
		userEntity *user.User
		pftDomain  *prefecture.Prefecture
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
		DeletedAt:      userEntity.DeletedAt(),
	}, nil
}

// CountUsers は、ユーザーの総件数を返すユースケースです。
func (u *usecase) CountUsers(ctx context.Context, active *bool) (int64, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()
	return u.userRepo.CountByActive(ctx, active)
}
