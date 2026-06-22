// Package backoff は、指数バックオフの待機時間を算出するユーティリティです。
// attempt を受け取り待機時間を返す純関数で、現在時刻や乱数には依存しません。
package backoff

import (
	"math"
	"time"
)

// Exponential は、指数バックオフの設定です。
type Exponential struct {
	// Initial は、attempt=0 のときの待機時間です。
	Initial time.Duration
	// Max は、待機時間の上限です（0 以下なら上限なし）。
	Max time.Duration
	// Multiplier は、attempt ごとの倍率です（1 未満は 1 として扱います）。
	Multiplier float64
}

// Duration は、attempt 回目（0 起算）の待機時間を返します。
// Initial * Multiplier^attempt を計算し、Max が正なら Max で上限を設けます。
func (e Exponential) Duration(attempt int) time.Duration {
	if e.Initial <= 0 {
		return 0
	}
	if attempt < 0 {
		attempt = 0
	}

	mult := e.Multiplier
	if mult < 1 {
		mult = 1
	}

	d := float64(e.Initial)
	for range attempt {
		d *= mult
		if e.Max > 0 && d >= float64(e.Max) {
			return e.Max
		}
	}
	if e.Max > 0 && d > float64(e.Max) {
		return e.Max
	}
	// Max 上限なし（Max<=0）かつ高 attempt で d が +Inf / int64 範囲外になりうる。
	// time.Duration(+Inf) は負値になり cooldown が即発火するため、MaxInt64 で頭打ちにする。
	if d > float64(math.MaxInt64) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(d)
}
