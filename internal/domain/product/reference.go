package product

import (
	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// StatusRef は、商品ステータスの ID と名称の組を表す値オブジェクトです。
// name は id に対応する表示専用の値で、呼び出し側での名称の別解決は不要です。
type StatusRef struct {
	id   uuid.UUID
	name string
}

// CategoryRef は、商品カテゴリの ID と名称の組を表す値オブジェクトです。
// name は id に対応する表示専用の値で、呼び出し側での名称の別解決は不要です。
type CategoryRef struct {
	id   uuid.UUID
	name string
}

// NewStatusRef は、商品ステータス参照の検証と生成を行います。
// id が nil、name が長さ制約（minRefNameLength〜maxRefNameLength）外の場合は検証エラーを返します。
func NewStatusRef(id uuid.UUID, name string) (StatusRef, error) {
	if id.IsNil() {
		return StatusRef{}, xerrors.Wrap(ErrInvalidStatusID, "statusID is required")
	}
	if ok, msg := stringkit.ValidateInRange(name, minRefNameLength, maxRefNameLength); !ok {
		return StatusRef{}, xerrors.Wrap(ErrInvalidStatusName, msg)
	}
	return StatusRef{id: id, name: name}, nil
}

// ID は、商品ステータス ID を返します。
func (s StatusRef) ID() uuid.UUID { return s.id }

// Name は、商品ステータス名を返します。
func (s StatusRef) Name() string { return s.name }

// NewCategoryRef は、商品カテゴリ参照の検証と生成を行います。
// id が nil、name が長さ制約（minRefNameLength〜maxRefNameLength）外の場合は検証エラーを返します。
func NewCategoryRef(id uuid.UUID, name string) (CategoryRef, error) {
	if id.IsNil() {
		return CategoryRef{}, xerrors.Wrap(ErrInvalidCategoryID, "categoryID is required")
	}
	if ok, msg := stringkit.ValidateInRange(name, minRefNameLength, maxRefNameLength); !ok {
		return CategoryRef{}, xerrors.Wrap(ErrInvalidCategoryName, msg)
	}
	return CategoryRef{id: id, name: name}, nil
}

// ID は、商品カテゴリ ID を返します。
func (c CategoryRef) ID() uuid.UUID { return c.id }

// Name は、商品カテゴリ名を返します。
func (c CategoryRef) Name() string { return c.name }
