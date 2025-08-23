// Package paging は、ページングの共通処理のための変換構造を提供します。
package paging

import (
	"fmt"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/pkg/xerrors"
)

type Paging struct {
	limit  int
	offset int
}

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

// NewPagingFrom1Based は、ページ番号と1ページあたりの件数から取得上限とオフセットを計算してPageを返します。
//
//	ページ番号が1未満の場合は1に、1ページあたりの件数が0以下の場合はデフォルト値を使用します。
//	perPage[limit]は、0以下の場合はdefaultPerPage値：最大値をmaxPerPageまで許容します。
//	page[offsetの係数]は、最小値1：最大値はmaxPageまで許容します。
func NewPagingFrom1Based(page, perPage *int) (Paging, error) {
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
		return Paging{}, xerrors.Wrap(apperror.ErrInvalidArgument, fmt.Sprintf("invalid page number: %d, max page is %d", pg, maxPage))
	}

	return Paging{
		limit:  limit,
		offset: (pg - 1) * limit,
	}, nil
}

// Limit は、ページの取得上限を返します。
func (p Paging) Limit() int { return p.limit }

// Offset は、ページのオフセットを返します。
func (p Paging) Offset() int { return p.offset }
