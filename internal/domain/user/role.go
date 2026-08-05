package user

import (
	"fmt"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// RoleCodeAdmin は、管理者ロールのコードです。
	RoleCodeAdmin RoleCode = 1
	// RoleCodeGeneral は、一般ロールのコードです。
	RoleCodeGeneral RoleCode = 2
)

// RoleCode は、ロールを識別するコード値です。永続化されている値であり、既知のコード以外は
// NewRole が ErrInvalidRoleCode で弾きます。
type RoleCode int

// Roles は、Role エンティティのスライス型です。
type Roles []*Role

// Role は、ユーザーに割り当てられるロールを表すドメインエンティティです。
type Role struct {
	id   uuid.UUID
	name string
	code RoleCode
}

// valid は、既知のロールコードかどうかを返します。
func (c RoleCode) valid() bool {
	switch c {
	case RoleCodeAdmin, RoleCodeGeneral:
		return true
	default:
		return false
	}
}

// HasAdmin は、管理者ロールを含むかどうかを返します。
func (rs Roles) HasAdmin() bool {
	for _, r := range rs {
		if r.IsAdmin() {
			return true
		}
	}

	return false
}

// NewRole は、ロールエンティティの検証と生成を行います。id が nil の場合は ErrInvalidRoleID、
// 名前長違反の場合は ErrInvalidRoleName、未知の code の場合は ErrInvalidRoleCode を返します。
func NewRole(
	id uuid.UUID,
	name string,
	code int,
) (*Role, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidRoleID, "id is required")
	}
	if ok, msg := stringkit.ValidateInRange(name, minRoleNameLength, maxRoleNameLength); !ok {
		return nil, xerrors.Wrap(ErrInvalidRoleName, msg)
	}
	rc := RoleCode(code)
	if !rc.valid() {
		return nil, xerrors.Wrap(ErrInvalidRoleCode, fmt.Sprintf("unknown role code: %d", code))
	}

	return &Role{
		id:   id,
		name: name,
		code: rc,
	}, nil
}

// ID は、ロールのIDを返します。
func (r *Role) ID() uuid.UUID { return r.id }

// Name は、ロール名を返します。
func (r *Role) Name() string { return r.name }

// Code は、ロールコードを返します。
func (r *Role) Code() RoleCode { return r.code }

// IsAdmin は、管理者ロールかどうかを返します。
func (r *Role) IsAdmin() bool { return r.code == RoleCodeAdmin }
