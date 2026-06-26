package retry

import (
	"math/rand/v2"
	"time"
)

// Full は、[0, d] の一様乱数を返します（full jitter）。
// バックオフに重畳して thundering herd を避けます。d が 0 以下なら 0 を返します。
func Full(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d) + 1)) //nolint:gosec // jitter は暗号強度不要
}
