// Package prefecture は、都道府県関連のドメインを提供します。
package prefecture

import (
	"fmt"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	MinPrefectureNameLength = 1
	MaxPrefectureNameLength = 100
	MinCode                 = 1
	MaxCode                 = 47
)

type Prefectures []*Prefecture

type Prefecture struct {
	id   uuid.UUID
	name string
	code int
}

// New は、都道府県エンティティの検証と生成を行います。
func New(
	id uuid.UUID,
	name string,
	code int,
) (*Prefecture, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
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
