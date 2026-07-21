// Package productstatus は、商品ステータスマスタドメインを定義します。ID・名称・コード・表示順（sortKey）の不変条件を持つ ProductStatus エンティティと Repository インターフェースを提供します。
package productstatus

import (
	"fmt"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// ProductStatuses は、ProductStatus エンティティのスライス型です。
type ProductStatuses []*ProductStatus

// ProductStatus は、商品ステータスを表すドメインエンティティです。
type ProductStatus struct {
	id      uuid.UUID
	name    string
	code    int
	sortKey int
}

// New は、商品ステータスエンティティの検証と生成を行います。code・sortKey は 1〜32767（正の SMALLINT）
// の整数である必要があります。id が nil の場合は ErrInvalidID、名前長・code 範囲・sortKey 範囲の
// 違反の場合は ErrInvalidName / ErrInvalidCode / ErrInvalidSortKey を返します。
func New(
	id uuid.UUID,
	name string,
	code int,
	sortKey int,
) (*ProductStatus, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if ok, msg := stringkit.ValidateInRange(name, minNameLength, maxNameLength); !ok {
		return nil, xerrors.Wrap(ErrInvalidName, msg)
	}
	if code < minCode || maxCode < code {
		return nil, xerrors.Wrap(
			ErrInvalidCode,
			fmt.Sprintf("code must be between %d and %d, got %d", minCode, maxCode, code),
		)
	}
	if sortKey < minSortKey || maxSortKey < sortKey {
		return nil, xerrors.Wrap(
			ErrInvalidSortKey,
			fmt.Sprintf("sort key must be between %d and %d, got %d", minSortKey, maxSortKey, sortKey),
		)
	}

	return &ProductStatus{
		id:      id,
		name:    name,
		code:    code,
		sortKey: sortKey,
	}, nil
}

// ID は、商品ステータスのIDを返します。
func (p *ProductStatus) ID() uuid.UUID { return p.id }

// Name は、商品ステータス名を返します。
func (p *ProductStatus) Name() string { return p.name }

// Code は、商品ステータスコードを返します。
func (p *ProductStatus) Code() int { return p.code }

// SortKey は、商品ステータスの表示順を返します。
func (p *ProductStatus) SortKey() int { return p.sortKey }
