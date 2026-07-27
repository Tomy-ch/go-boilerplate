package conv

import "time"

// TimeOrZero は、任意指定の日時（*time.Time）をレスポンスの time.Time へ変換します。
// nil の場合はゼロ値へ倒します（値が必ず存在する状況で防御的に nil を扱う用途を想定）。
func TimeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
