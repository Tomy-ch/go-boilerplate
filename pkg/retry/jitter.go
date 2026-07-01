package retry

import (
	"math"
	"math/rand/v2"
	"time"
)

// Full は、[0, d] の一様乱数を返します（full jitter）。
// バックオフに重畳して thundering herd を避けます。d が 0 以下なら 0 を返します。
//
// ただし d == math.MaxInt64（バックオフ上限到達時に起こり得る）のときは、閉区間化のための
// 加算が int64 をオーバーフローするため、上限 d を除いた [0, d) を返します。
func Full(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	n := int64(d)
	if n != math.MaxInt64 {
		n++
	}
	return time.Duration(rand.Int64N(n)) //nolint:gosec // jitter は暗号強度不要
}
