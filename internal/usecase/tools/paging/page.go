// Package paging は、ページングの共通処理のための変換構造を提供します。
package paging

import (
	"fmt"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
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
	// int32 変換時のオフセット上限（int32 境界の防御）
	maxOffset = maxPage * maxPerPage
)

// Page は、ページネーションに必要な取得上限（limit）とオフセット（offset）を保持する値オブジェクトです。
type Page struct {
	limit  int
	offset int
}

// NewPageFrom1Based は、ページ番号と1ページあたりの件数から取得上限とオフセットを計算して Page を返します。
//
//	ページ番号が1未満の場合は1に、1ページあたりの件数が0以下の場合はデフォルト値を使用します。
//	page[offsetの係数]は最小値1で、maxPage を超える場合はエラーを返します。
//	perPage[limit]は、0以下の場合はdefaultPerPage値：最大値をmaxPerPageまで許容します。
func NewPageFrom1Based(page, perPage *int) (*Page, error) {
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

	return &Page{
		limit:  limit,
		offset: (pg - 1) * limit,
	}, nil
}

// Limit は、ページの取得上限を返します。
func (p Page) Limit() int { return p.limit }

// Limit32 は、ページの取得上限をint32型で返します。
func (p Page) Limit32() int32 {
	limit := min(p.limit, maxPerPage)
	//nolint:gosec // G115: maxPerPage(int32範囲内の定数)でクランプ済みのためオーバーフローしません
	return int32(limit)
}

// Offset は、ページのオフセットを返します。
func (p Page) Offset() int { return p.offset }

// Offset32 は、ページのオフセットをint32型で返します。
func (p Page) Offset32() int32 {
	offset := min(p.offset, maxOffset)
	//nolint:gosec // G115: maxOffset(int32範囲内の定数)でクランプ済みのためオーバーフローしません
	return int32(offset)
}
