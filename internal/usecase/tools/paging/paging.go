// Package paging は、ページングの共通処理のための変換構造を提供します。
package paging

import (
	"fmt"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/pkg/xerrors"
)

const (
	// 1ページあたりのデフォルト件数
	defaultPerPage = 50
	// 1ページあたりの最大件数
	maxPerPage = 200
	// 最小ページ番号
	minPage = 1
	// サーバが許容する最大ページ数
	maxPage = 10_000
)

type Paging struct {
	limit  int
	offset int
}

// NewPagingFrom1Based は、ページ番号と1ページあたりの件数から取得上限とオフセットを計算してPageを返します。
//
//	ページ番号が1未満の場合は1に、1ページあたりの件数が0以下の場合はデフォルト値を使用します。
//	page[offsetの係数]は、最小値1：最大値はmaxPageまで許容します。
//	perPage[limit]は、0以下の場合はdefaultPerPage値：最大値をmaxPerPageまで許容します。
func NewPagingFrom1Based(page, perPage *int) (*Paging, error) {
	limit := defaultPerPage
	if perPage != nil && *perPage > 0 {
		limit = *perPage
	}
	if limit > maxPerPage {
		limit = maxPerPage
	}

	pg := minPage
	if page != nil && *page > 0 {
		pg = *page
	}
	if pg > maxPage {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, fmt.Sprintf("invalid page number: %d, max page is %d", pg, maxPage))
	}

	return &Paging{
		limit:  limit,
		offset: (pg - 1) * limit,
	}, nil
}

// Limit は、ページの取得上限を返します。
func (p Paging) Limit() int { return p.limit }

// Limit32 は、ページの取得上限をint32型で返します。
// limit は1ページあたりの件数であり、安全のためmaxPerPageでクランプしてからint32へ変換します。
func (p Paging) Limit32() int32 {
	limit := p.limit
	if limit > maxPerPage {
		limit = maxPerPage
	}
	return int32(limit)
}

// Offset は、ページのオフセットを返します。
func (p Paging) Offset() int { return p.offset }

// Offset32 は、ページのオフセットをint32型で返します。
// offset が maxOffset を超える場合でも、maxOffset にクランプした上で int32 に変換することで安全性を担保します。
func (p Paging) Offset32() int32 {
	offset := p.offset
	maxOffset := maxPage * maxPerPage
	if offset > maxOffset {
		offset = maxOffset
	}
	return int32(offset)
}
