//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package search は、ユーザー検索に関するユースケースを提供します。
package search

import (
	"context"

	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/tools/paging"
	"boilerplate-go/internal/usecase/tools/search"
	"boilerplate-go/internal/usecase/user/search/query"
)

// SearchParams は、ユーザー検索のパラメータを表します。
type SearchParams struct {
	Keyword *string
	Active  *bool
}

// usecase は、ユーザー検索を提供するクエリサービスを提供します。
type usecase struct {
	tracer observability.LayerTracer
	userQS query.UserQueryService
}

// Usecase は、ユーザーに関するユースケースを定義します。
type Usecase interface {
	// ListUsersByKeyword は、キーワードに基づいてユーザー一覧を取得します。
	ListUsersByKeyword(ctx context.Context, filter *SearchParams, page *paging.Paging) (query.UserSearchResults, error)
	// CountUsersByKeyword は、キーワードに基づいてユーザーの総件数を返します。
	CountUsersByKeyword(ctx context.Context, filter *SearchParams) (int64, error)
}

// New は、ユーザーに関するユースケースを初期化します。
func New(
	tf observability.TracerFactory,
	userQueryService query.UserQueryService,
) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		userQS: userQueryService,
	}
}

// ListUsersByKeyword は、ユーザー一覧を取得するユースケースです。
func (u *usecase) ListUsersByKeyword(ctx context.Context, filter *SearchParams, page *paging.Paging) (query.UserSearchResults, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	keywords := search.ParseSearchTokens(filter.Keyword, search.DefaultMaxTokens)
	searchFilter := &query.UserSearchFilter{
		Active:   filter.Active,
		Keywords: keywords,
	}
	return u.userQS.FindByFilter(ctx, searchFilter, page.Limit32(), page.Offset32())
}

// CountUsersByKeyword は、キーワードに基づいてユーザーの総件数を返すユースケースです。
func (u *usecase) CountUsersByKeyword(ctx context.Context, filter *SearchParams) (int64, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	keywords := search.ParseSearchTokens(filter.Keyword, search.DefaultMaxTokens)
	searchFilter := &query.UserSearchFilter{
		Active:   filter.Active,
		Keywords: keywords,
	}
	return u.userQS.CountByFilter(ctx, searchFilter)
}
