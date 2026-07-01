// Package user は、ユーザーに関するクエリーサービスを提供します。
package user

import (
	"context"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user/search/query"
)

type service struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、ユーザー検索クエリサービスの実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.UserSearchQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// buildLikeTokens は、キーワードを LIKE パターンへ変換します。空の場合は全件マッチの ["%"] を返します。
func buildLikeTokens(keywords []string) []string {
	if len(keywords) == 0 {
		return []string{"%"}
	}
	tokens := make([]string, len(keywords))
	for i, kw := range keywords {
		escaped := sqlc.EscapeForLike(kw, sqlc.DefaultLikeEscapeChar)
		tokens[i] = sqlc.WrapContainsLikePattern(escaped)
	}
	return tokens
}

// toUserSearchResult は、sqlc の行から UserSearchResult を構築します。
func toUserSearchResult(u gen.Users, prefectureName string) *query.UserSearchResult {
	return &query.UserSearchResult{
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		Email:          u.Email,
		Phone:          u.Phone,
		PostalCode:     u.PostalCode,
		PrefectureName: prefectureName,
		City:           u.City,
		Street:         u.Street,
		Building:       u.Building,
		RegisteredAt:   u.CreatedAt,
		DeletedAt:      u.DeletedAt,
	}
}

// FindByFilter は、キーワード検索でユーザーの情報を取得します。
func (s *service) FindByFilter(ctx context.Context, filter *query.UserSearchFilter, limit, offset int32) (query.UserSearchResults, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	tokens := buildLikeTokens(filter.Keywords)
	db := gen.New(driver.New(ctx, s.db))

	switch {
	case filter.Active == nil:
		return fetchSearchAll(ctx, db, &gen.SearchUsersParams{
			PatternsParam: tokens,
			LimitParam:    limit,
			OffsetParam:   offset,
		})
	case *filter.Active:
		return fetchSearchActive(ctx, db, &gen.SearchActiveUsersParams{
			PatternsParam: tokens,
			LimitParam:    limit,
			OffsetParam:   offset,
		})
	default:
		return fetchSearchDeleted(ctx, db, &gen.SearchDeletedUsersParams{
			PatternsParam: tokens,
			LimitParam:    limit,
			OffsetParam:   offset,
		})
	}
}

// fetchSearchAll は、キーワード検索で全てのユーザーを検索します。
func fetchSearchAll(
	ctx context.Context, db *gen.Queries, searchParams *gen.SearchUsersParams,
) (query.UserSearchResults, error) {
	rows, err := db.SearchUsers(ctx, searchParams)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	results := make(query.UserSearchResults, len(rows))
	for i, row := range rows {
		results[i] = toUserSearchResult(row.Users, row.PrefectureName)
	}
	return results, nil
}

// fetchSearchActive は、アクティブなユーザーを検索します。
func fetchSearchActive(
	ctx context.Context, db *gen.Queries, searchParams *gen.SearchActiveUsersParams,
) (query.UserSearchResults, error) {
	rows, err := db.SearchActiveUsers(ctx, searchParams)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	results := make(query.UserSearchResults, len(rows))
	for i, row := range rows {
		results[i] = toUserSearchResult(row.Users, row.PrefectureName)
	}
	return results, nil
}

// fetchSearchDeleted は、削除されたユーザーを検索します。
func fetchSearchDeleted(
	ctx context.Context, db *gen.Queries, searchParams *gen.SearchDeletedUsersParams,
) (query.UserSearchResults, error) {
	rows, err := db.SearchDeletedUsers(ctx, searchParams)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	results := make(query.UserSearchResults, len(rows))
	for i, row := range rows {
		results[i] = toUserSearchResult(row.Users, row.PrefectureName)
	}
	return results, nil
}

// CountByFilter は、キーワード検索でユーザーの総件数を返します。
func (s *service) CountByFilter(ctx context.Context, filter *query.UserSearchFilter) (int64, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	tokens := buildLikeTokens(filter.Keywords)
	db := gen.New(driver.New(ctx, s.db))

	var (
		count int64
		err   error
	)

	switch {
	case filter.Active == nil:
		count, err = db.CountSearchUsers(ctx, tokens)
	case *filter.Active:
		count, err = db.CountSearchActiveUsers(ctx, tokens)
	default:
		count, err = db.CountSearchDeletedUsers(ctx, tokens)
	}

	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}

	return count, nil
}
