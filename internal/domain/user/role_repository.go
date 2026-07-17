//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package user

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// RoleRepository は、ユーザーに割り当てられたロールの取得を定義するドメインリポジトリインターフェースです。
type RoleRepository interface {
	// FindRolesByUserID は、指定ユーザーに割り当てられた全ロールを取得します。
	// 割り当てが存在しない場合は空のスライスを返します（NotFound は返しません）。
	FindRolesByUserID(ctx context.Context, userID uuid.UUID) (Roles, error)
}
