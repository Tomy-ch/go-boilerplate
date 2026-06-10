//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package user は、ユーザーに関するユースケースを提供します。
package user

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/security"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// UserView は、ユーザー取得結果の出力 DTO を表します。
type UserView struct {
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

// UpdateProfileParams は、ユーザープロフィール更新の入力（可変フィールド）を表します。
type UpdateProfileParams struct {
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

// CreateParamsDTO は、ユーザー作成に必要なパラメータを表します。
type CreateParamsDTO struct {
	UserID      uuid.UUID
	RawPassword string

	UpdateProfileParams
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
	encrypter security.Hasher
	userRepo  user.Repository
	pftRepo   prefecture.Repository
}

// Usecase は、ユーザーに関するユースケースを定義します。
type Usecase interface {
	// ListUsers は、ユーザー一覧を取得します。
	ListUsers(ctx context.Context, active *bool, page *paging.Paging) ([]UserView, error)
	// CreateUser は、ユーザーを作成します。
	CreateUser(ctx context.Context, dto *CreateParamsDTO) (UserView, error)
	// CountUsers は、ユーザーの総件数を返します。
	CountUsers(ctx context.Context, active *bool) (int64, error)
	// GetUser は、IDから単一ユーザーを取得します。
	GetUser(ctx context.Context, id uuid.UUID) (UserView, error)
	// UpdateUser は、ユーザーのプロフィールを全更新します（パスワードは含みません）。
	UpdateUser(ctx context.Context, id uuid.UUID, dto *UpdateProfileParams) (UserView, error)
	// UpdateUserPartially は、ユーザーを部分更新します（パスワードは更新しません）。
	UpdateUserPartially(ctx context.Context, id uuid.UUID, dto *PatchParamsDTO) (UserView, error)
	// ChangePassword は、現在のパスワードを照合したうえでユーザーのパスワードを変更します。
	ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) error
	// DeleteUser は、ユーザーを論理削除します。
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

// New は、ユーザーに関するユースケースを初期化します。
func New(
	tf observability.TracerFactory,
	txm tx.Manager,
	clock clock.Clock,
	encrypter security.Hasher,
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

func (u *usecase) ListUsers(ctx context.Context, active *bool, page *paging.Paging) ([]UserView, error) {
	if page == nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "page must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	us, err := u.userRepo.FindByActive(ctx, active, page.Limit32(), page.Offset32())
	if err != nil {
		return nil, err
	}

	_, prefectureMap, err := observability.RunWithSpan(
		ctx, u.tracer, "usecase", "user", "prefectureMap", func(ctx context.Context) (map[uuid.UUID]*prefecture.Prefecture, error) {
			pids := make([]uuid.UUID, len(us))
			for i, ue := range us {
				pids[i] = ue.PrefectureID()
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

	dtos := make([]UserView, len(us))
	for i, ue := range us {
		p, ok := prefectureMap[ue.PrefectureID()]
		if !ok {
			return nil, xerrors.Wrap(apperror.ErrNotFound, "prefecture not found for user")
		}
		dtos[i] = toUserView(ue, p.Name())
	}

	return dtos, nil
}

// CreateUser は、ユーザーを作成するユースケースです。
func (u *usecase) CreateUser(ctx context.Context, dto *CreateParamsDTO) (UserView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()
	rawPassword, err := user.NewRawPassword(dto.RawPassword)
	if err != nil {
		return UserView{}, err
	}

	passwordHash, err := u.encrypter.Hash(rawPassword.Value())
	if err != nil {
		return UserView{}, err
	}

	var (
		userEntity *user.User
		pftName    string
	)
	err = u.txm.Do(ctx, func(ctx context.Context) error {
		pftDomain, err := u.pftRepo.FindByName(ctx, dto.PrefectureName)
		if err != nil {
			return err
		}
		pftName = pftDomain.Name()

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

		return u.userRepo.Create(ctx, userEntity)
	})
	if err != nil {
		return UserView{}, err
	}

	return toUserView(userEntity, pftName), nil
}

// CountUsers は、ユーザーの総件数を返すユースケースです。
func (u *usecase) CountUsers(ctx context.Context, active *bool) (int64, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()
	return u.userRepo.CountByActive(ctx, active)
}

// GetUser は、IDから単一ユーザーを取得するユースケースです。
func (u *usecase) GetUser(ctx context.Context, id uuid.UUID) (UserView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	userEntity, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}

	pftDomain, err := u.pftRepo.FindByID(ctx, userEntity.PrefectureID())
	if err != nil {
		return UserView{}, err
	}

	return toUserView(userEntity, pftDomain.Name()), nil
}

// UpdateUser は、ユーザーのプロフィールを全更新するユースケースです（PUT、パスワードは含みません）。
func (u *usecase) UpdateUser(ctx context.Context, id uuid.UUID, dto *UpdateProfileParams) (UserView, error) {
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

		pftDomain, err := u.pftRepo.FindByName(ctx, dto.PrefectureName)
		if err != nil {
			return err
		}
		pftName = pftDomain.Name()

		if err = userEntity.UpdateProfile(
			dto.FirstName,
			dto.LastName,
			dto.Email,
			dto.Phone,
			pftDomain.ID(),
			dto.City,
			dto.Street,
			dto.Building,
			dto.PostalCode,
			now,
		); err != nil {
			return err
		}

		return u.userRepo.Update(ctx, userEntity)
	})
	if err != nil {
		return UserView{}, err
	}

	return toUserView(userEntity, pftName), nil
}

// ChangePassword は、現在のパスワードを照合したうえでユーザーのパスワードを変更するユースケースです。
func (u *usecase) ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) error {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()
	// 現パスワードも newPassword と同じ長さ制約で検証する（bcrypt の 72 バイト切り詰めを避け、入力の対称性を保つ）。
	if _, err := user.NewRawPassword(currentPassword); err != nil {
		return err
	}
	rawNew, err := user.NewRawPassword(newPassword)
	if err != nil {
		return err
	}

	return u.txm.Do(ctx, func(ctx context.Context) error {
		userEntity, err := u.userRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		matched, err := u.encrypter.Compare(userEntity.PasswordHash(), currentPassword)
		if err != nil {
			return err
		}
		if !matched {
			return user.ErrCurrentPasswordMismatch
		}

		newHash, err := u.encrypter.Hash(rawNew.Value())
		if err != nil {
			return err
		}

		if err = userEntity.ChangePassword(newHash, now); err != nil {
			return err
		}

		return u.userRepo.Update(ctx, userEntity)
	})
}

// UpdateUserPartially は、ユーザーを部分更新するユースケースです（PATCH）。
// 指定フィールドのみ更新し、未指定/null は据え置く（クリアは非対応。クリアは PUT を使う）。password は更新しない。
func (u *usecase) UpdateUserPartially(ctx context.Context, id uuid.UUID, dto *PatchParamsDTO) (UserView, error) {
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

		if err = userEntity.UpdateProfile(
			ptr.Deref(dto.FirstName, userEntity.FirstName()),
			ptr.Deref(dto.LastName, userEntity.LastName()),
			ptr.Deref(dto.Email, userEntity.Email()),
			ptr.Deref(dto.Phone, userEntity.Phone()),
			prefectureID,
			ptr.Deref(dto.City, userEntity.City()),
			ptr.Deref(dto.Street, userEntity.Street()),
			building,
			ptr.Deref(dto.PostalCode, userEntity.PostalCode()),
			now,
		); err != nil {
			return err
		}
		return u.userRepo.Update(ctx, userEntity)
	})
	if err != nil {
		return UserView{}, err
	}

	return toUserView(userEntity, pftName), nil
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

// toUserView は、ユーザーエンティティと都道府県名から DTO を構築します。
func toUserView(u *user.User, prefectureName string) UserView {
	return UserView{
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
