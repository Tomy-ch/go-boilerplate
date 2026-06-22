// Package backoff は、指数バックオフの待機時間を算出する framework-agnostic なユーティリティです。
// 純関数として attempt を受け取り、現在時刻や乱数に依存しません（決定的・テスト容易）。
package backoff

import "time"

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
	return time.Duration(d)
}
