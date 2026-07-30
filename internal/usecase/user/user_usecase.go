//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package user は、ユーザーに関するユースケースを提供します。
package user

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/user/event"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// aggregateType は、outbox の集約種別です。
const aggregateType = "user"

// 既存ユーザーが参照する prefecture を解決できない参照整合性破れ（サーバ側データ不整合）を表します。
var errOrphanPrefecture = xerrors.Wrap(apperror.ErrInternal, "prefecture not found for user")

// 進行中の購入が残っているユーザーの退会要求を表します。
var errInProgressPurchaseExists = xerrors.Wrap(apperror.ErrConflict, "user has in-progress purchases")

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

// UserListView は、一覧取得結果（ページ分の一覧と総件数）を表します。
type UserListView struct {
	Items []UserView
	Total int64
}

// UserFeedView は、ユーザーフィード（cursor ページネーション）の取得結果を表します。
type UserFeedView struct {
	// Items は、現在ページのユーザー一覧です。
	Items []UserView
	// NextCursor は、次ページ取得用の不透明カーソルです。最終ページの場合は nil です。
	NextCursor *string
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
	UpdateProfileParams

	UserID uuid.UUID
}

// PatchParamsDTO は、ユーザー部分更新（PATCH）に必要なパラメータを表します。nil のフィールドは更新しません。
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
	tracer       observability.LayerTracer
	txm          tx.Manager
	clock        clock.Clock
	authorizer   authz.Authorizer
	userRepo     user.Repository
	pftRepo      prefecture.Repository
	purchaseRepo purchase.Repository
	emit         outbox.EmitUsecase
}

// Usecase は、ユーザーに関するユースケースを定義します。
type Usecase interface {
	// ListUsers は、ユーザー一覧を取得します。
	ListUsers(ctx context.Context, active *bool, page *paging.Page) ([]UserView, error)
	// ListUsersWithTotal は、ユーザー一覧と総件数をまとめて取得します。
	ListUsersWithTotal(ctx context.Context, active *bool, page *paging.Page) (*UserListView, error)
	// ListUsersFeed は、未削除ユーザーを作成日時の降順（cursor ページネーション）で取得します。
	ListUsersFeed(ctx context.Context, cursor *paging.Cursor) (*UserFeedView, error)
	// CreateUser は、ユーザーを作成します。
	CreateUser(ctx context.Context, dto *CreateParamsDTO) (UserView, error)
	// CountUsers は、ユーザーの総件数を返します。
	CountUsers(ctx context.Context, active *bool) (int64, error)
	// GetUser は、認可を確認したうえで ID から単一ユーザーを取得します。
	// 認可が拒否された場合は authz.ErrForbidden（apperror.ErrPermissionDenied をラップ）を返します。
	GetUser(ctx context.Context, authn *authbd.Authn, id uuid.UUID) (UserView, error)
	// UpdateUser は、認可を確認したうえでユーザーのプロフィールを全更新します。
	// 認可が拒否された場合は authz.ErrForbidden（apperror.ErrPermissionDenied をラップ）を返します。
	UpdateUser(ctx context.Context, authn *authbd.Authn, id uuid.UUID, dto *UpdateProfileParams) (UserView, error)
	// UpdateUserPartially は、認可を確認したうえでユーザーを部分更新します。指定されたフィールドのみを
	// 反映し、未指定 / null は据え置きます（値のクリアは非対応で、クリアには全更新の UpdateUser を使います）。
	// 認可が拒否された場合は authz.ErrForbidden（apperror.ErrPermissionDenied をラップ）を返します。
	UpdateUserPartially(ctx context.Context, authn *authbd.Authn, id uuid.UUID, dto *PatchParamsDTO) (UserView, error)
	// DeleteUser は、認可を確認したうえでユーザーを退会させます。ユーザーを論理削除し、
	// 同一トランザクションで退会イベントを発行します。退会に伴う関連データの後始末は、
	// このイベントを受け取る側の結果整合に委ねます。
	// 進行中の購入が残っている場合は apperror.ErrConflict を返し、退会させません。
	// 認可が拒否された場合は authz.ErrForbidden（apperror.ErrPermissionDenied をラップ）を返します。
	DeleteUser(ctx context.Context, authn *authbd.Authn, id uuid.UUID) error
}

// New は、ユーザーに関するユースケースを初期化します。
func New(
	tf observability.TracerFactory,
	txm tx.Manager,
	clock clock.Clock,
	authorizer authz.Authorizer,
	userRepo user.Repository,
	prefectureRepo prefecture.Repository,
	purchaseRepo purchase.Repository,
	emit outbox.EmitUsecase,
) Usecase {
	return &usecase{
		tracer:       tf.Usecase(),
		txm:          txm,
		clock:        clock,
		authorizer:   authorizer,
		userRepo:     userRepo,
		pftRepo:      prefectureRepo,
		purchaseRepo: purchaseRepo,
		emit:         emit,
	}
}

func (u *usecase) ListUsers(ctx context.Context, active *bool, page *paging.Page) ([]UserView, error) {
	if page == nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "page must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	us, err := u.userRepo.FindByActive(ctx, active, page.Limit32(), page.Offset32())
	if err != nil {
		return nil, err
	}

	return u.toUserViews(ctx, us)
}

func (u *usecase) CreateUser(ctx context.Context, dto *CreateParamsDTO) (UserView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()

	var (
		userEntity *user.User
		pftName    string
	)
	err := u.txm.Do(ctx, func(ctx context.Context) error {
		pftDomain, err := u.pftRepo.FindByName(ctx, dto.PrefectureName)
		if err != nil {
			return err
		}
		pftName = pftDomain.Name()

		userEntity, err = user.New(dto.UserID, user.Attributes{
			Profile: user.Profile{
				FirstName:    dto.FirstName,
				LastName:     dto.LastName,
				Email:        dto.Email,
				Phone:        dto.Phone,
				PrefectureID: pftDomain.ID(),
				City:         dto.City,
				Street:       dto.Street,
				Building:     dto.Building,
				PostalCode:   dto.PostalCode,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
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

func (u *usecase) CountUsers(ctx context.Context, active *bool) (int64, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()
	return u.userRepo.CountByActive(ctx, active)
}

func (u *usecase) ListUsersWithTotal(ctx context.Context, active *bool, page *paging.Page) (*UserListView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	items, err := u.ListUsers(ctx, active, page)
	if err != nil {
		return nil, err
	}
	total, err := u.CountUsers(ctx, active)
	if err != nil {
		return nil, err
	}
	return &UserListView{Items: items, Total: total}, nil
}

func (u *usecase) ListUsersFeed(ctx context.Context, cursor *paging.Cursor) (*UserFeedView, error) {
	if cursor == nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "cursor must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	after, err := decodeFeedCursor(cursor)
	if err != nil {
		return nil, err
	}

	us, err := u.userRepo.FindFeed(ctx, after, cursor.Limit32()+1)
	if err != nil {
		return nil, err
	}

	limit := cursor.Limit()
	hasNext := len(us) > limit
	if hasNext {
		us = us[:limit]
	}

	items, err := u.toUserViews(ctx, us)
	if err != nil {
		return nil, err
	}

	var nextCursor *string
	if hasNext && len(us) > 0 {
		encoded := encodeFeedCursor(us[len(us)-1])
		nextCursor = &encoded
	}

	return &UserFeedView{Items: items, NextCursor: nextCursor}, nil
}

func (u *usecase) GetUser(ctx context.Context, authn *authbd.Authn, id uuid.UUID) (UserView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorizeUserAccess(ctx, authn, authz.ActionUserGet, id); err != nil {
		return UserView{}, err
	}

	userEntity, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}

	pftDomain, err := u.pftRepo.FindByID(ctx, userEntity.PrefectureID())
	if err != nil {
		if xerrors.Is(err, apperror.ErrNotFound) {
			return UserView{}, errOrphanPrefecture
		}
		return UserView{}, err
	}

	return toUserView(userEntity, pftDomain.Name()), nil
}

func (u *usecase) UpdateUser(ctx context.Context, authn *authbd.Authn, id uuid.UUID, dto *UpdateProfileParams) (UserView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorizeUserAccess(ctx, authn, authz.ActionUserUpdate, id); err != nil {
		return UserView{}, err
	}

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

		if err = userEntity.UpdateProfile(user.Profile{
			FirstName:    dto.FirstName,
			LastName:     dto.LastName,
			Email:        dto.Email,
			Phone:        dto.Phone,
			PrefectureID: pftDomain.ID(),
			City:         dto.City,
			Street:       dto.Street,
			Building:     dto.Building,
			PostalCode:   dto.PostalCode,
		}, now); err != nil {
			return err
		}

		return u.userRepo.Update(ctx, userEntity)
	})
	if err != nil {
		return UserView{}, err
	}

	return toUserView(userEntity, pftName), nil
}

func (u *usecase) UpdateUserPartially(ctx context.Context, authn *authbd.Authn, id uuid.UUID, dto *PatchParamsDTO) (UserView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorizeUserAccess(ctx, authn, authz.ActionUserUpdate, id); err != nil {
		return UserView{}, err
	}

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

		prefectureID := userEntity.PrefectureID()
		pftDomain, err := u.resolvePatchPrefecture(ctx, dto.PrefectureName, prefectureID)
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

		if err = userEntity.UpdateProfile(user.Profile{
			FirstName:    ptr.Deref(dto.FirstName, userEntity.FirstName()),
			LastName:     ptr.Deref(dto.LastName, userEntity.LastName()),
			Email:        ptr.Deref(dto.Email, userEntity.Email()),
			Phone:        ptr.Deref(dto.Phone, userEntity.Phone()),
			PrefectureID: prefectureID,
			City:         ptr.Deref(dto.City, userEntity.City()),
			Street:       ptr.Deref(dto.Street, userEntity.Street()),
			Building:     building,
			PostalCode:   ptr.Deref(dto.PostalCode, userEntity.PostalCode()),
		}, now); err != nil {
			return err
		}
		return u.userRepo.Update(ctx, userEntity)
	})
	if err != nil {
		return UserView{}, err
	}

	return toUserView(userEntity, pftName), nil
}

func (u *usecase) DeleteUser(ctx context.Context, authn *authbd.Authn, id uuid.UUID) error {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorizeUserAccess(ctx, authn, authz.ActionUserDelete, id); err != nil {
		return err
	}

	now := u.clock.Now()
	// 論理削除と退会イベントの発行を単一 tx にまとめ、退会だけが成立してイベントが失われることを防ぐ。
	// 退会を拒む条件（進行中の購入）も同じ tx で判定し、拒否時は論理削除もイベントも残さない。
	return u.txm.Do(ctx, func(ctx context.Context) error {
		userEntity, err := u.userRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		inProgress, err := u.purchaseRepo.ExistsInProgressByUserID(ctx, id)
		if err != nil {
			return err
		}
		if inProgress {
			return errInProgressPurchaseExists
		}

		if err = userEntity.MarkAsDeleted(now); err != nil {
			return err
		}
		if err = u.userRepo.Update(ctx, userEntity); err != nil {
			return err
		}

		payload, err := event.BuildWithdrawn(userEntity)
		if err != nil {
			return err
		}
		_, err = u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   id.String(),
			EventType:     event.TypeWithdrawn,
			Payload:       payload,
		})
		return err
	})
}

// authorizeUserAccess は、認証主体 authn が対象ユーザー（所有者 = id）への action を実行してよいか判定します。
// リソースの所有者を対象ユーザー（id）とすることで、所有権モデルでは呼出元が自分自身のみを操作できます。
// authn が nil の場合は、認可判定以前に呼出元を特定できないため apperror.ErrUnauthenticated を返します。
func (u *usecase) authorizeUserAccess(ctx context.Context, authn *authbd.Authn, action authz.Action, id uuid.UUID) error {
	if authn == nil {
		return apperror.ErrUnauthenticated
	}
	return u.authorizer.Authorize(ctx, authn, action, authz.NewResource("user", &id))
}

// toUserViews は、ユーザーエンティティ列を UserView の DTO 列へ変換します。
// いずれかのユーザーが参照する都道府県を解決できない場合は参照整合性破れ（errOrphanPrefecture）を返します。
func (u *usecase) toUserViews(ctx context.Context, us user.Users) ([]UserView, error) {
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
			return nil, errOrphanPrefecture
		}
		dtos[i] = toUserView(ue, p.Name())
	}

	return dtos, nil
}

// resolvePatchPrefecture は、部分更新時の都道府県を解決します。
// 名前指定があれば名前で解決（入力エラーは伝播）、なければ既存 ID で解決します（未解決は参照整合性破れ）。
func (u *usecase) resolvePatchPrefecture(ctx context.Context, name *string, currentID uuid.UUID) (*prefecture.Prefecture, error) {
	if name != nil {
		return u.pftRepo.FindByName(ctx, *name)
	}
	pftDomain, err := u.pftRepo.FindByID(ctx, currentID)
	if xerrors.Is(err, apperror.ErrNotFound) {
		return nil, errOrphanPrefecture
	}
	if err != nil {
		return nil, err
	}
	return pftDomain, nil
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
