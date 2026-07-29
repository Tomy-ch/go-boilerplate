// Package event は、ユーザーの outbox イベント payload を提供します。
package event

import (
	"encoding/json"
	"time"

	"go-boilerplate/internal/domain/user"
	"go-boilerplate/pkg/xerrors"
)

// TypeWithdrawn は、ユーザー退会の outbox イベント種別（version 込み）です。
const TypeWithdrawn = "user.withdrawn.v1"

// withdrawn は、user.withdrawn.v1 の自己完結 snapshot payload です。
type withdrawn struct {
	UserID    string `json:"userId"`
	DeletedAt string `json:"deletedAt"`
}

// BuildWithdrawn は、ユーザー集約から user.withdrawn.v1 の自己完結 snapshot payload を marshal します。
// 退会していないユーザーを渡した場合、payload の deletedAt は空文字列になります。
func BuildWithdrawn(u *user.User) ([]byte, error) {
	var deletedAt string
	if at := u.DeletedAt(); at != nil {
		deletedAt = at.Format(time.RFC3339Nano)
	}

	payload, err := json.Marshal(withdrawn{
		UserID:    u.ID().String(),
		DeletedAt: deletedAt,
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to encode user.withdrawn payload")
	}
	return payload, nil
}
