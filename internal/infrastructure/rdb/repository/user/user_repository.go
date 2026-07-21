// Package user は、ユーザーに関するドメインのリポジトリを提供します。
package user

import (
	"context"

	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

type repository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、user.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) user.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindByActive は、アクティブ状態に基づいてユーザーの情報を取得します。
func (r *repository) FindByActive(ctx context.Context, active *bool, limit, offset int32) (user.Users, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	switch {
	case active == nil:
		return fetchListUsersRows(ctx, db, &gen.ListUsersParams{
			OffsetParam: offset,
			LimitParam:  limit,
		})
	case *active:
		return fetchListUsersRowsByActive(ctx, db, &gen.ListActiveUsersParams{
			OffsetParam: offset,
			LimitParam:  limit,
		})
	default:
		return fetchListUsersRowsByDeleted(ctx, db, &gen.ListDeletedUsersParams{
			OffsetParam: offset,
			LimitParam:  limit,
		})
	}
}

// FindFeed は、未削除ユーザーを (created_at DESC, id DESC) の安定順で keyset ページネーション取得します。
// after=nil の場合は先頭ページ、それ以外は after が表す境界より後ろ（より過去）の行を返します。
func (r *repository) FindFeed(ctx context.Context, after *user.FeedCursor, limit int32) (user.Users, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	if after == nil {
		rows, err := db.ListUsersFeedFirst(ctx, limit)
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToUsers(rows, func(r *gen.ListUsersFeedFirstRow) gen.Users { return r.Users })
	}

	rows, err := db.ListUsersFeedAfter(ctx, &gen.ListUsersFeedAfterParams{
		AfterCreatedAt: after.CreatedAt(),
		AfterID:        after.ID(),
		LimitParam:     limit,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return rowsToUsers(rows, func(r *gen.ListUsersFeedAfterRow) gen.Users { return r.Users })
}

// SearchByKeyword は、検索テキストがいずれかのキーワードに部分一致するユーザーを、作成日時の降順でページング取得します。
// active=nil で全件、true でアクティブのみ、false で削除済みのみを対象とします。keywords が空の場合は全ユーザーを対象とします。
func (r *repository) SearchByKeyword(ctx context.Context, keywords []string, active *bool, limit, offset int32) (user.Users, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	tokens := buildLikeTokens(keywords)
	db := gen.New(driver.New(ctx, r.db))

	switch {
	case active == nil:
		rows, err := db.SearchUsers(ctx, &gen.SearchUsersParams{
			PatternsParam: tokens,
			LimitParam:    limit,
			OffsetParam:   offset,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToUsers(rows, func(r *gen.SearchUsersRow) gen.Users { return r.Users })
	case *active:
		rows, err := db.SearchActiveUsers(ctx, &gen.SearchActiveUsersParams{
			PatternsParam: tokens,
			LimitParam:    limit,
			OffsetParam:   offset,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToUsers(rows, func(r *gen.SearchActiveUsersRow) gen.Users { return r.Users })
	default:
		rows, err := db.SearchDeletedUsers(ctx, &gen.SearchDeletedUsersParams{
			PatternsParam: tokens,
			LimitParam:    limit,
			OffsetParam:   offset,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToUsers(rows, func(r *gen.SearchDeletedUsersRow) gen.Users { return r.Users })
	}
}

// buildLikeTokens は、キーワードを部分一致の LIKE パターンへ変換します。空の場合は全件マッチの ["%"] を返します。
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

// rowToUser は、sqlc が返す Users 行をドメインエンティティへ変換します。
// 再構築時の検証失敗はデータ不整合として ErrInternal へ正規化します（422 / details にしない）。
func rowToUser(u gen.Users) (*user.User, error) {
	entity, err := user.New(
		u.ID,
		u.FirstName,
		u.LastName,
		u.PasswordHash,
		u.Email,
		u.Phone,
		u.PrefectureID,
		u.City,
		u.Street,
		u.Building,
		u.PostalCode,
		u.CreatedAt,
		u.UpdatedAt,
		u.DeletedAt,
	)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// rowsToUsers は、行スライスをドメインエンティティ列へ変換します。
func rowsToUsers[T any](rows []T, extract func(T) gen.Users) (user.Users, error) {
	users := make(user.Users, len(rows))
	for i, row := range rows {
		u, err := rowToUser(extract(row))
		if err != nil {
			return nil, err
		}
		users[i] = u
	}
	return users, nil
}

// fetchListUsersRows は、ユーザーの情報を取得します。
func fetchListUsersRows(
	ctx context.Context, db *gen.Queries, params *gen.ListUsersParams,
) (user.Users, error) {
	rows, err := db.ListUsers(ctx, params)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return rowsToUsers(rows, func(r *gen.ListUsersRow) gen.Users { return r.Users })
}

// fetchListUsersRowsByActive は、アクティブ状態に基づいてユーザーの情報を取得します。
func fetchListUsersRowsByActive(
	ctx context.Context, db *gen.Queries, params *gen.ListActiveUsersParams,
) (user.Users, error) {
	rows, err := db.ListActiveUsers(ctx, params)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return rowsToUsers(rows, func(r *gen.ListActiveUsersRow) gen.Users { return r.Users })
}

// fetchListUsersRowsByDeleted は、削除されたユーザーの情報を取得します。
func fetchListUsersRowsByDeleted(
	ctx context.Context, db *gen.Queries, params *gen.ListDeletedUsersParams,
) (user.Users, error) {
	rows, err := db.ListDeletedUsers(ctx, params)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return rowsToUsers(rows, func(r *gen.ListDeletedUsersRow) gen.Users { return r.Users })
}

// Create は、ユーザーを作成します。
func (r *repository) Create(ctx context.Context, u *user.User) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	err := db.CreateUser(ctx, &gen.CreateUserParams{
		ID:           u.ID(),
		FirstName:    u.FirstName(),
		LastName:     u.LastName(),
		PasswordHash: u.PasswordHash(),
		Email:        u.Email(),
		Phone:        u.Phone(),
		PrefectureID: u.PrefectureID(),
		City:         u.City(),
		Street:       u.Street(),
		Building:     u.Building(),
		PostalCode:   u.PostalCode(),
		CreatedAt:    u.CreatedAt(),
		UpdatedAt:    u.UpdatedAt(),
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// FindByID は、IDから単一ユーザーを取得します。存在しない場合は NotFound に正規化したエラーを返します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetUserByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToUser(row.Users)
}

// Update は、ユーザーの mutable フィールドと updatedAt / deletedAt を更新します。
func (r *repository) Update(ctx context.Context, u *user.User) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.UpdateUser(ctx, &gen.UpdateUserParams{
		FirstName:    u.FirstName(),
		LastName:     u.LastName(),
		PasswordHash: u.PasswordHash(),
		Email:        u.Email(),
		Phone:        u.Phone(),
		PrefectureID: u.PrefectureID(),
		City:         u.City(),
		Street:       u.Street(),
		Building:     u.Building(),
		PostalCode:   u.PostalCode(),
		UpdatedAt:    u.UpdatedAt(),
		DeletedAt:    u.DeletedAt(),
		ID:           u.ID(),
	})
	// エラー正規化と「影響行数 0 → NotFound」判定は pgerror に集約
	return pgerror.NormalizeExecResult(rows, err)
}

// CountByActive は、アクティブ状態に基づいてユーザーの総件数を返します。
func (r *repository) CountByActive(ctx context.Context, active *bool) (int64, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	var (
		count int64
		err   error
	)

	switch {
	case active == nil:
		count, err = db.CountUsers(ctx)
	case *active:
		count, err = db.CountActiveUsers(ctx)
	default:
		count, err = db.CountDeletedUsers(ctx)
	}

	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}

	return count, nil
}

// CountByKeyword は、検索テキストがいずれかのキーワードに部分一致するユーザーの総件数を返します。
// active / keywords の意味は SearchByKeyword と同じです。
func (r *repository) CountByKeyword(ctx context.Context, keywords []string, active *bool) (int64, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	tokens := buildLikeTokens(keywords)
	db := gen.New(driver.New(ctx, r.db))

	var (
		count int64
		err   error
	)

	switch {
	case active == nil:
		count, err = db.CountSearchUsers(ctx, tokens)
	case *active:
		count, err = db.CountSearchActiveUsers(ctx, tokens)
	default:
		count, err = db.CountSearchDeletedUsers(ctx, tokens)
	}

	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}

	return count, nil
}
