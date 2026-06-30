package retry

import (
	"math"
	"math/rand/v2"
	"time"
)

// Full は、[0, d] の一様乱数を返します（full jitter）。
// バックオフに重畳して thundering herd を避けます。d が 0 以下なら 0 を返します。
//
// d が math.MaxInt64（バックオフの上限到達時に発生し得る）の場合、[0, d] の閉区間に
// するための +1 が int64 をオーバーフローするため、上限値 1 点のみ除外した [0, d) を返します。
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
