package paging

import "math"

// LimitPolicy は、取得件数の補完・クランプ規約を表します。
//
//	フィールド名で意味を固定するため、既定値と上限を取り違えたまま組み立てることができません。
type LimitPolicy struct {
	// Default は、件数が指定されなかった場合に採用する件数です。
	Default int
	// Max は、許容する最大件数です。これを超える要求はこの値へ切り詰めます。
	Max int
}

// Limit は、ポリシーで正規化済みの取得件数を表す値オブジェクトです。
//
//	ページングを持つ読み取り（Page / Cursor）と、ページングを持たない top-N 読み取り
//	（ランキング・在庫僅少一覧など）の双方が、この型を通して同じ件数ポリシーを共有します。
//	top-N はページ番号もカーソルも持たないため、件数だけを Page / Cursor から切り離してこの型が担います。
type Limit struct {
	value int
}

// NewLimit は、要求件数をポリシーで正規化した Limit を返します。
//
//	first が nil または 0 以下の場合は policy.Default を、policy.Max を超える場合は policy.Max を採用します。
func NewLimit(first *int, policy LimitPolicy) Limit {
	value := policy.Default
	if first != nil && *first > 0 {
		value = *first
	}
	if value > policy.Max {
		value = policy.Max
	}

	return Limit{value: value}
}

// Value は、正規化後の取得件数を返します。
func (l Limit) Value() int { return l.value }

// Value32 は、正規化後の取得件数をint32型で返します。
func (l Limit) Value32() int32 {
	//nolint:gosec // G115: math.MaxInt32 でクランプ済みのためオーバーフローしません
	return int32(min(l.value, math.MaxInt32))
}
