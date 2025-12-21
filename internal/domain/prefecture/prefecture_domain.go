// Package prefecture は、都道府県関連のドメインを提供します。
package prefecture

import (
	"fmt"

	"boilerplate-go/pkg/stringkit"
	"boilerplate-go/pkg/uuid"
	"boilerplate-go/pkg/xerrors"
)

const (
	MinPrefectureNameLength = 1
	MaxPrefectureNameLength = 100
	MinCode                 = 1
	MaxCode                 = 47
)

type Entities []*Entity

type Entity struct {
	id   uuid.UUID
	name string
	code int
}

// New は、都道府県エンティティの検証と生成を行います。
func New(
	idStr string,
	name string,
	code int,
) (*Entity, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, xerrors.Wrap(ErrInvalidID, err.Error())
	}
	if !stringkit.InRange(name, MinPrefectureNameLength, MaxPrefectureNameLength) {
		return nil, xerrors.Wrap(
			ErrInvalidPrefectureName,
			stringkit.ErrorMsgInRange(MinPrefectureNameLength, MaxPrefectureNameLength, name),
		)
	}
	if code < MinCode || MaxCode < code {
		return nil, xerrors.Wrap(
			ErrInvalidCode,
			fmt.Sprintf("code must be between %d and %d, got %d", MinCode, MaxCode, code),
		)
	}

	return &Entity{
		id:   id,
		name: name,
		code: code,
	}, nil
}

// ID は、都道府県のIDを返します。
func (p *Entity) ID() uuid.UUID { return p.id }

// Name は、都道府県名を返します。
func (p *Entity) Name() string { return p.name }

// Code は、都道府県コードを返します。
func (p *Entity) Code() int { return p.code }
