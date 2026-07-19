package metrics

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_queryMetrics_Observe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ErrorClassが空の場合はdurationのみ記録しerror counterを増やさない", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			recorder := NewQueryRecorder(reg)

			recorder.Observe(context.Background(), driver.QueryAttrs{
				QueryName: "user.find_by_id",
				Operation: "select",
				Status:    "success",
				Duration:  time.Millisecond,
			})

			duration := findMetricFamily(t, reg, "rdb_query_duration_seconds")
			require.NotNil(t, duration)
			require.Len(t, duration.GetMetric(), 1)
			assert.Equal(t, uint64(1), duration.GetMetric()[0].GetHistogram().GetSampleCount())

			assert.Nil(t, findMetricFamily(t, reg, "rdb_query_errors_total"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ErrorClassが非空の場合はerror counterをerror_class付きで増分する", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			recorder := NewQueryRecorder(reg)

			recorder.Observe(context.Background(), driver.QueryAttrs{
				QueryName:  "user.create",
				Operation:  "insert",
				Status:     "error",
				ErrorClass: "constraint",
				Duration:   time.Millisecond,
			})

			errs := findMetricFamily(t, reg, "rdb_query_errors_total")
			require.NotNil(t, errs)
			require.Len(t, errs.GetMetric(), 1)
			assert.InDelta(t, 1.0, errs.GetMetric()[0].GetCounter().GetValue(), 0)
			assert.Equal(t, "constraint", labelValue(errs.GetMetric()[0], "error_class"))
		})
	})
}
