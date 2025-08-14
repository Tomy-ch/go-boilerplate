// Package userrepo は、ユーザーに関するドメインのリポジトリを提供します。
package userrepo

import (
	"context"
	"database/sql"

	"boilerplate-go/internal/infrastructure/rdb"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/repository/user/gen"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) GetAllUsers(ctx context.Context) ([]gen.GetUsersDomainRow, error) {
	return gen.New(rdb.ResolveConn(ctx, r.db)).GetUsersDomain(ctx, gen.GetUsersDomainParams{})
}
