//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package query は、ユーザーに関するクエリサービスを提供します。
package query

import (
	"context"
	"time"
)

type UserQueryService interface {
	// FindByFilter は、キーワード検索でユーザーの情報を取得します。
	FindByFilter(ctx context.Context, filter *UserSearchFilter, limit, offset int32) (UserSearchResults, error)
	// CountByFilter は、キーワード検索でユーザーの総件数を返します。
	CountByFilter(ctx context.Context, filter *UserSearchFilter) (int64, error)
}

// UserSearchFilter は、ユーザー検索のフィルタリング条件を表します。
type UserSearchFilter struct {
	Active   *bool
	Keywords []string
}

// UserSearchResults は、ユーザー検索の結果を表します。
type UserSearchResults []*UserSearchResult

// UserSearchResult は、ユーザー検索の結果を表します。
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
