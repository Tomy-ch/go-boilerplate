// Package user は、ユーザーに関するクエリーサービスを提供します。
package user

import (
	"context"

	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/infrastructure/rdb/postgres/pgerror"
	"boilerplate-go/internal/infrastructure/rdb/sqlc"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/user/search/query"
)

type service struct {
	db     loggingdb.DBProvider
	tracer observability.LayerTracer
}

func New(
	db loggingdb.DBProvider,
	tf observability.TracerFactory,
) query.UserQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindByKeyword は、キーワード検索でユーザーの情報を取得します。
func (s *service) FindByFilter(ctx context.Context, filter *query.UserSearchFilter, limit, offset int32) (query.UserSearchResults, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	tokens := make([]string, len(filter.Keywords))
	for i, kw := range filter.Keywords {
		escaped := sqlc.EscapeForLike(kw, sqlc.DefaultLikeEscapeChar)
		tokens[i] = sqlc.WrapContainsLikePattern(escaped)
	}

	db := gen.New(s.db.NewLoggingDB(ctx))

	switch {
	case filter.Active == nil:
		return fetchSearchAll(ctx, db, &gen.SearchUsersParams{
			PatternsParam: tokens,
			LimitParam:    int32(limit),
			OffsetParam:   int32(offset),
		})
	case *filter.Active:
		return fetchSearchActive(ctx, db, &gen.SearchActiveUsersParams{
			PatternsParam: tokens,
			LimitParam:    int32(limit),
			OffsetParam:   int32(offset),
		})
	case !*filter.Active:
		return fetchSearchDeleted(ctx, db, &gen.SearchDeletedUsersParams{
			PatternsParam: tokens,
			LimitParam:    int32(limit),
			OffsetParam:   int32(offset),
		})
	default:
		panic("unreachable: invalid active")
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
		results[i] = &query.UserSearchResult{
			FirstName:      row.Users.FirstName,
			LastName:       row.Users.LastName,
			Email:          row.Users.Email,
			Phone:          row.Users.Phone,
			PostalCode:     row.Users.PostalCode,
			PrefectureName: row.PrefecqtureName,
			City:           row.Users.City,
			Street:         row.Users.Street,
			Building:       row.Users.Building,
			RegisteredAt:   row.Users.CreatedAt,
			DeletedAt:      row.Users.DeletedAt,
		}
	}
	return results, err
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
		results[i] = &query.UserSearchResult{
			FirstName:      row.Users.FirstName,
			LastName:       row.Users.LastName,
			Email:          row.Users.Email,
			Phone:          row.Users.Phone,
			PostalCode:     row.Users.PostalCode,
			PrefectureName: row.PrefecqtureName,
			City:           row.Users.City,
			Street:         row.Users.Street,
			Building:       row.Users.Building,
			RegisteredAt:   row.Users.CreatedAt,
			DeletedAt:      row.Users.DeletedAt,
		}
	}
	return results, err
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
		results[i] = &query.UserSearchResult{
			FirstName:      row.Users.FirstName,
			LastName:       row.Users.LastName,
			Email:          row.Users.Email,
			Phone:          row.Users.Phone,
			PostalCode:     row.Users.PostalCode,
			PrefectureName: row.PrefecqtureName,
			City:           row.Users.City,
			Street:         row.Users.Street,
			Building:       row.Users.Building,
			RegisteredAt:   row.Users.CreatedAt,
			DeletedAt:      row.Users.DeletedAt,
		}
	}
	return results, err
}

// CountByFilter は、キーワード検索でユーザーの総件数を返します。
func (s *service) CountByFilter(ctx context.Context, filter *query.UserSearchFilter) (int64, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	tokens := make([]string, len(filter.Keywords))
	for i, kw := range filter.Keywords {
		escaped := sqlc.EscapeForLike(kw, sqlc.DefaultLikeEscapeChar)
		tokens[i] = sqlc.WrapContainsLikePattern(escaped)
	}

	db := gen.New(s.db.NewLoggingDB(ctx))

	var (
		count int64
		err   error
	)

	switch {
	case filter.Active == nil:
		count, err = db.CountSearchUsers(ctx, tokens)
	case *filter.Active:
		count, err = db.CountSearchActiveUsers(ctx, tokens)
	case !*filter.Active:
		count, err = db.CountSearchDeletedUsers(ctx, tokens)
	default:
		panic("unreachable: invalid active")
	}

	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}

	return count, nil
}
