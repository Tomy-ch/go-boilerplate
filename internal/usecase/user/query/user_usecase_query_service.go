//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package query は、ユーザーに関するクエリサービスを提供します。
package query

import (
	"context"

	"boilerplate-go/internal/domain/user"
)

type UserQueryService interface {
	// FindByKeyword は、キーワード検索でユーザーの情報を取得します。
	FindByKeyword(ctx context.Context, keywords []string, active *bool, limit, offset int32) (user.Users, error)
}
