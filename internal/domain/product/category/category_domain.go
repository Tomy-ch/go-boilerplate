// Package category は、商品カテゴリドメインを定義します。名称・コード・表示順（sortKey）の
// 不変条件を持つ Category エンティティと Repository インターフェースを提供します。
package category

import (
	"fmt"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Categories は、Category エンティティのスライス型です。
type Categories []*Category

// Category は、商品カテゴリを表すドメインエンティティです。
type Category struct {
	id      uuid.UUID
	name    string
	code    int
	sortKey int
}

// Attributes は、商品カテゴリの属性一式です。Code と SortKey は同じ int かつ同じ値域（1〜32767）で、
// 位置引数のままだと取り違えても検証を通過してしまうため構造体で受けます。
type Attributes struct {
	Name    string
	Code    int
	SortKey int
}

// New は、商品カテゴリエンティティの検証と生成を行います。Code・SortKey は 1〜32767 の整数である
// 必要があります。id が nil の場合は ErrInvalidID、名前長・Code 範囲・SortKey 範囲の違反の場合は
// それぞれ ErrInvalidName / ErrInvalidCode / ErrInvalidSortKey を返します。
func New(id uuid.UUID, attrs Attributes) (*Category, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if ok, msg := stringkit.ValidateInRange(attrs.Name, minNameLength, maxNameLength); !ok {
		return nil, xerrors.Wrap(ErrInvalidName, msg)
	}
	if attrs.Code < minCode || maxCode < attrs.Code {
		return nil, xerrors.Wrap(
			ErrInvalidCode,
			fmt.Sprintf("code must be between %d and %d, got %d", minCode, maxCode, attrs.Code),
		)
	}
	if attrs.SortKey < minSortKey || maxSortKey < attrs.SortKey {
		return nil, xerrors.Wrap(
			ErrInvalidSortKey,
			fmt.Sprintf("sort key must be between %d and %d, got %d", minSortKey, maxSortKey, attrs.SortKey),
		)
	}

	return &Category{
		id:      id,
		name:    attrs.Name,
		code:    attrs.Code,
		sortKey: attrs.SortKey,
	}, nil
}

// ID は、商品カテゴリのIDを返します。
func (c *Category) ID() uuid.UUID { return c.id }

// Name は、商品カテゴリ名を返します。
func (c *Category) Name() string { return c.name }

// Code は、商品カテゴリコードを返します。
func (c *Category) Code() int { return c.code }

// SortKey は、商品カテゴリの表示順を返します。
func (c *Category) SortKey() int { return c.sortKey }
