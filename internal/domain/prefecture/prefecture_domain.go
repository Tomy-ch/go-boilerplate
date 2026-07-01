// Package prefecture は、都道府県ドメインを定義します。JIS コード（1〜47）・名称の不変条件を持つ Prefecture エンティティと Repository インターフェースを提供します。
package prefecture

import (
	"fmt"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Prefectures は、Prefecture エンティティのスライス型です。
type Prefectures []*Prefecture

// Prefecture は、都道府県を表すドメインエンティティです。
type Prefecture struct {
	id   uuid.UUID
	name string
	code int
}

// New は、都道府県エンティティの検証と生成を行います。code は 1〜47（JIS 都道府県コード）の
// 整数である必要があります。id が nil の場合は ErrInvalidID、名前長または code 範囲違反の場合は
// ErrInvalidName / ErrInvalidCode を返します。
func New(
	id uuid.UUID,
	name string,
	code int,
) (*Prefecture, error) {
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

	return &Prefecture{
		id:   id,
		name: name,
		code: code,
	}, nil
}

// ID は、都道府県のIDを返します。
func (p *Prefecture) ID() uuid.UUID { return p.id }

// Name は、都道府県名を返します。
func (p *Prefecture) Name() string { return p.name }

// Code は、都道府県コードを返します。
func (p *Prefecture) Code() int { return p.code }
