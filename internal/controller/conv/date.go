package conv

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// DatePtr は、任意指定の日付クエリパラメータを *time.Time へ変換します。
// nil の場合はそのまま nil を返します。openapi_types.Date は暦日のみを表す検証済みの値型で、
// 時刻成分は持たないため、変換は無条件に成功します。
func DatePtr(p *openapi_types.Date) *time.Time {
	if p == nil {
		return nil
	}
	t := p.Time
	return &t
}
