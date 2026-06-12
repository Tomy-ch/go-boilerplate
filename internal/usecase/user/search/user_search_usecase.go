//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package search は、ユーザー検索に関するユースケースを提供します。
package search

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/tools/search"
	"go-boilerplate/internal/usecase/user/search/query"
	"go-boilerplate/pkg/xerrors"
)

// SearchParams は、ユーザー検索のパラメータを表します。
type SearchParams struct {
	Keyword *string
	Active  *bool
}

// UserSearchListView は、検索結果（ページ分の一覧と総件数）を表します。
type UserSearchListView struct {
	Items query.UserSearchResults
	Total int64
}

// usecase は、Usecase インターフェースの実装です。
type usecase struct {
	tracer observability.LayerTracer
	userQS query.UserSearchQueryService
}

// Usecase は、ユーザーに関するユースケースを定義します。
type Usecase interface {
	// ListUsersByKeyword は、キーワードに基づいてユーザー一覧を取得します。
	ListUsersByKeyword(ctx context.Context, filter *SearchParams, page *paging.Paging) (query.UserSearchResults, error)
	// CountUsersByKeyword は、キーワードに基づいてユーザーの総件数を返します。
	CountUsersByKeyword(ctx context.Context, filter *SearchParams) (int64, error)
	// ListUsersByKeywordWithTotal は、検索一覧と総件数をまとめて取得します。
	ListUsersByKeywordWithTotal(ctx context.Context, filter *SearchParams, page *paging.Paging) (*UserSearchListView, error)
}

// New は、ユーザーに関するユースケースを初期化します。
func New(
	tf observability.TracerFactory,
	userQueryService query.UserSearchQueryService,
) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		userQS: userQueryService,
	}
}

// ListUsersByKeyword は、ユーザー一覧を取得するユースケースです。
func (u *usecase) ListUsersByKeyword(ctx context.Context, filter *SearchParams, page *paging.Paging) (query.UserSearchResults, error) {
	if filter == nil || page == nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "filter and page must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	return u.userQS.FindByFilter(ctx, u.toFilter(filter), page.Limit32(), page.Offset32())
}

// CountUsersByKeyword は、キーワードに基づいてユーザーの総件数を返すユースケースです。
func (u *usecase) CountUsersByKeyword(ctx context.Context, filter *SearchParams) (int64, error) {
	if filter == nil {
		return 0, xerrors.Wrap(apperror.ErrInvalidArgument, "filter must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	return u.userQS.CountByFilter(ctx, u.toFilter(filter))
}

// ListUsersByKeywordWithTotal は、一覧と総件数の合成を controller から usecase へ寄せる。
func (u *usecase) ListUsersByKeywordWithTotal(ctx context.Context, filter *SearchParams, page *paging.Paging) (*UserSearchListView, error) {
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

// toFilter は、SearchParams から検索フィルタを構築します。
func (u *usecase) toFilter(p *SearchParams) *query.UserSearchFilter {
	return &query.UserSearchFilter{
		Active:   p.Active,
		Keywords: search.ParseSearchTokens(p.Keyword, search.DefaultMaxTokens),
	}
}
