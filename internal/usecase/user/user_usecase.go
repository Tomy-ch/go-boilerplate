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

// UpdateParamsDTO は、ユーザー全更新（PUT）に必要なパラメータを表します。全フィールド必須で password も更新します。
type UpdateParamsDTO struct {
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	PostalCode     string
	PrefectureName string
	City           string
	Street         string
	Building       *string
	RawPassword    string
}

// PatchParamsDTO は、ユーザー部分更新（PATCH）に必要なパラメータを表します。nil のフィールドは更新しません（password は更新対象外）。
type PatchParamsDTO struct {
	FirstName      *string
	LastName       *string
	Email          *string
	Phone          *string
	PostalCode     *string
	PrefectureName *string
	City           *string
	Street         *string
	Building       *string
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
	// GetUser は、IDから単一ユーザーを取得します。
	GetUser(ctx context.Context, id uuid.UUID) (MutableFields, error)
	// UpdateUser は、ユーザーを全更新します（パスワードも更新）。
	UpdateUser(ctx context.Context, id uuid.UUID, dto *UpdateParamsDTO) (MutableFields, error)
	// UpdateUserPartially は、ユーザーを部分更新します（パスワードは更新しません）。
	UpdateUserPartially(ctx context.Context, id uuid.UUID, dto *PatchParamsDTO) (MutableFields, error)
	// DeleteUser は、ユーザーを論理削除します。
	DeleteUser(ctx context.Context, id uuid.UUID) error
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
		if p, ok := prefectureMap[u.PrefectureID()]; ok {
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

// GetUser は、IDから単一ユーザーを取得するユースケースです。
func (u *usecase) GetUser(ctx context.Context, id uuid.UUID) (MutableFields, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	userEntity, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return MutableFields{}, err
	}

	pftDomain, err := u.pftRepo.FindByID(ctx, userEntity.PrefectureID())
	if err != nil {
		return MutableFields{}, err
	}

	return toMutableFields(userEntity, pftDomain.Name()), nil
}

// UpdateUser は、ユーザーを全更新するユースケースです（PUT、パスワードも更新）。
func (u *usecase) UpdateUser(ctx context.Context, id uuid.UUID, dto *UpdateParamsDTO) (MutableFields, error) {
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
		userEntity, err = u.userRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		pftDomain, err = u.pftRepo.FindByName(ctx, dto.PrefectureName)
		if err != nil {
			return err
		}

		if err = userEntity.UpdateProfile(
			dto.FirstName,
			dto.LastName,
			dto.Email,
			dto.Phone,
			pftDomain.ID(),
			dto.PostalCode,
			dto.City,
			dto.Street,
			dto.Building,
			now,
		); err != nil {
			return err
		}

		if err = userEntity.ChangePassword(passwordHash, now); err != nil {
			return err
		}

		return u.userRepo.Update(ctx, userEntity)
	})
	if err != nil {
		return MutableFields{}, err
	}

	return toMutableFields(userEntity, pftDomain.Name()), nil
}

// UpdateUserPartially は、ユーザーを部分更新するユースケースです（PATCH）。
// 指定フィールドのみ更新し、未指定/null は据え置く（クリアは非対応。クリアは PUT を使う）。password は更新しない。
func (u *usecase) UpdateUserPartially(ctx context.Context, id uuid.UUID, dto *PatchParamsDTO) (MutableFields, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()

	var (
		userEntity *user.User
		pftName    string
	)
	err := u.txm.Do(ctx, func(ctx context.Context) error {
		var err error
		userEntity, err = u.userRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		// 都道府県: 指定があれば名前解決、なければ現在の prefecture を取得（レスポンス名の解決も兼ねる）
		prefectureID := userEntity.PrefectureID()
		var pftDomain *prefecture.Prefecture
		if dto.PrefectureName != nil {
			pftDomain, err = u.pftRepo.FindByName(ctx, *dto.PrefectureName)
		} else {
			pftDomain, err = u.pftRepo.FindByID(ctx, prefectureID)
		}
		if err != nil {
			return err
		}
		prefectureID = pftDomain.ID()
		pftName = pftDomain.Name()

		// provided なフィールドのみ現在値に上書きしたフルセットを構築
		building := userEntity.Building()
		if dto.Building != nil {
			building = dto.Building
		}

		return updateProfileThenSave(ctx, u.userRepo, userEntity,
			derefOr(dto.FirstName, userEntity.FirstName()),
			derefOr(dto.LastName, userEntity.LastName()),
			derefOr(dto.Email, userEntity.Email()),
			derefOr(dto.Phone, userEntity.Phone()),
			prefectureID,
			derefOr(dto.PostalCode, userEntity.PostalCode()),
			derefOr(dto.City, userEntity.City()),
			derefOr(dto.Street, userEntity.Street()),
			building,
			now,
		)
	})
	if err != nil {
		return MutableFields{}, err
	}

	return toMutableFields(userEntity, pftName), nil
}

// DeleteUser は、ユーザーを論理削除するユースケースです（DELETE）。
func (u *usecase) DeleteUser(ctx context.Context, id uuid.UUID) error {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()
	return u.txm.Do(ctx, func(ctx context.Context) error {
		userEntity, err := u.userRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err = userEntity.MarkAsDeleted(now); err != nil {
			return err
		}
		return u.userRepo.Update(ctx, userEntity)
	})
}

// updateProfileThenSave は、entity.UpdateProfile を適用してから Update で永続化します。
func updateProfileThenSave(
	ctx context.Context, repo user.Repository, userEntity *user.User,
	firstName, lastName, email, phone string,
	prefectureID uuid.UUID,
	postalCode, city, street string,
	building *string,
	now time.Time,
) error {
	if err := userEntity.UpdateProfile(
		firstName, lastName, email, phone, prefectureID, postalCode, city, street, building, now,
	); err != nil {
		return err
	}
	return repo.Update(ctx, userEntity)
}

// toMutableFields は、ユーザーエンティティと都道府県名から DTO を構築します。
func toMutableFields(u *user.User, prefectureName string) MutableFields {
	return MutableFields{
		FirstName:      u.FirstName(),
		LastName:       u.LastName(),
		Email:          u.Email(),
		Phone:          u.Phone(),
		PostalCode:     u.PostalCode(),
		PrefectureName: prefectureName,
		City:           u.City(),
		Street:         u.Street(),
		Building:       u.Building(),
		DeletedAt:      u.DeletedAt(),
	}
}

// derefOr は、p が非 nil ならその値、nil なら current を返します（PATCH のマージ用）。
func derefOr(p *string, current string) string {
	if p != nil {
		return *p
	}
	return current
}
