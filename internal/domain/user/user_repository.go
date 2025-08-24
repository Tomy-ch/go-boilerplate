//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock$GOPACKAGE
package user

import (
	"context"
)

type Repository interface {
	GetAllUsers(ctx context.Context, limit, offset int) (Entities, error)
}
