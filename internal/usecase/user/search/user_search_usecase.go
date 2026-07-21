//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package search は、ユーザー検索に関するユースケースを提供します。
package search

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/tools/search"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// 既存ユーザーが参照する prefecture を解決できない参照整合性破れ（サーバ側データ不整合）を表します。
var errOrphanPrefecture = xerrors.Wrap(apperror.ErrInternal, "prefecture not found for user")

// SearchParams は、ユーザー検索のパラメータを表します。
type SearchParams struct {
	Keyword *string
	Active  *bool
}

// UserSearchResults は、ユーザー検索結果の一覧（リスト）を表します。
type UserSearchResults []*UserSearchResult

// UserSearchResult は、ユーザー検索結果1件を表します。
type UserSearchResult struct {
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	PostalCode     string
	PrefectureName string
	City           string
	Street         string
	Building       *string
	RegisteredAt   time.Time
	DeletedAt      *time.Time
}

// UserSearchListView は、検索結果（ページ分の一覧と総件数）を表します。
type UserSearchListView struct {
	Items UserSearchResults
	Total int64
}

// usecase は、Usecase インターフェースの実装です。
type usecase struct {
	tracer   observability.LayerTracer
	userRepo user.Repository
	pftRepo  prefecture.Repository
}

// Usecase は、キーワード検索によるユーザー検索ユースケースを定義します。
type Usecase interface {
	// ListUsersByKeyword は、キーワードに基づいてユーザー一覧を取得します。
	ListUsersByKeyword(ctx context.Context, filter *SearchParams, page *paging.Page) (UserSearchResults, error)
	// CountUsersByKeyword は、キーワードに基づいてユーザーの総件数を返します。
	CountUsersByKeyword(ctx context.Context, filter *SearchParams) (int64, error)
	// ListUsersByKeywordWithTotal は、検索一覧と総件数をまとめて取得します。
	ListUsersByKeywordWithTotal(ctx context.Context, filter *SearchParams, page *paging.Page) (*UserSearchListView, error)
}

// New は、ユーザーに関するユースケースを初期化します。
func New(
	tf observability.TracerFactory,
	userRepo user.Repository,
	prefectureRepo prefecture.Repository,
) Usecase {
	return &usecase{
		tracer:   tf.Usecase(),
		userRepo: userRepo,
		pftRepo:  prefectureRepo,
	}
}

// ListUsersByKeyword は、キーワードに基づいてユーザー一覧を取得します。
func (u *usecase) ListUsersByKeyword(ctx context.Context, filter *SearchParams, page *paging.Page) (UserSearchResults, error) {
	if filter == nil || page == nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "filter and page must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	keywords := search.ParseSearchTokens(filter.Keyword, search.DefaultMaxTokens)
	us, err := u.userRepo.SearchByKeyword(ctx, keywords, filter.Active, page.Limit32(), page.Offset32())
	if err != nil {
		return nil, err
	}

	return u.toSearchResults(ctx, us)
}

// CountUsersByKeyword は、キーワードに基づいてユーザーの総件数を返すユースケースです。
func (u *usecase) CountUsersByKeyword(ctx context.Context, filter *SearchParams) (int64, error) {
	if filter == nil {
		return 0, xerrors.Wrap(apperror.ErrInvalidArgument, "filter must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	keywords := search.ParseSearchTokens(filter.Keyword, search.DefaultMaxTokens)
	return u.userRepo.CountByKeyword(ctx, keywords, filter.Active)
}

// ListUsersByKeywordWithTotal は、検索一覧と総件数をまとめて取得します。
func (u *usecase) ListUsersByKeywordWithTotal(ctx context.Context, filter *SearchParams, page *paging.Page) (*UserSearchListView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	items, err := u.ListUsersByKeyword(ctx, filter, page)
	if err != nil {
		return nil, err
	}
	total, err := u.CountUsersByKeyword(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &UserSearchListView{Items: items, Total: total}, nil
}

// toSearchResults は、ユーザーエンティティ列を UserSearchResult の DTO 列へ変換します。
// いずれかのユーザーが参照する都道府県を解決できない場合は参照整合性破れ（errOrphanPrefecture）を返します。
func (u *usecase) toSearchResults(ctx context.Context, us user.Users) (UserSearchResults, error) {
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

	results := make(UserSearchResults, len(us))
	for i, ue := range us {
		p, ok := prefectureMap[ue.PrefectureID()]
		if !ok {
			return nil, errOrphanPrefecture
		}
		results[i] = toSearchResult(ue, p.Name())
	}

	return results, nil
}

// toSearchResult は、ユーザーエンティティと都道府県名から検索結果 DTO を構築します。
func toSearchResult(u *user.User, prefectureName string) *UserSearchResult {
	return &UserSearchResult{
		FirstName:      u.FirstName(),
		LastName:       u.LastName(),
		Email:          u.Email(),
		Phone:          u.Phone(),
		PostalCode:     u.PostalCode(),
		PrefectureName: prefectureName,
		City:           u.City(),
		Street:         u.Street(),
		Building:       u.Building(),
		RegisteredAt:   u.CreatedAt(),
		DeletedAt:      u.DeletedAt(),
	}
}
