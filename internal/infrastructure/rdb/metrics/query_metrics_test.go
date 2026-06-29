package metrics

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findMetricFamily は、Gather 結果から指定名のメトリクスファミリを返します。
func findMetricFamily(t *testing.T, reg *prometheus.Registry, name string) *dto.MetricFamily {
	t.Helper()

	families, err := reg.Gather()
	require.NoError(t, err)

	for _, mf := range families {
		if mf.GetName() == name {
			return mf
		}
	}
	return nil
}

// labelValue は、メトリクスから指定ラベルの値を返します。
func labelValue(m *dto.Metric, name string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == name {
			return l.GetValue()
		}
	}
	return ""
}

func TestNewQueryRecorder(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("成功時はdurationを記録しerror counterは増やさない", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			recorder := NewQueryRecorder(reg)

			recorder.Observe(context.Background(), driver.QueryAttrs{
				QueryName: "user.find_by_id",
				Operation: "select",
				Status:    "success",
				Duration:  10 * time.Millisecond,
			})

			duration := findMetricFamily(t, reg, "rdb_query_duration_seconds")
			require.NotNil(t, duration)
			require.Len(t, duration.GetMetric(), 1)
			assert.Equal(t, uint64(1), duration.GetMetric()[0].GetHistogram().GetSampleCount())

			// 成功時は error counter のメトリクスが存在しない。
			errs := findMetricFamily(t, reg, "rdb_query_errors_total")
			assert.Nil(t, errs)
		})

		t.Run("durationラベルはquery_name/operation/statusのみ", func(t *testing.T) {
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
			labels := duration.GetMetric()[0].GetLabel()

			names := make([]string, 0, len(labels))
			for _, l := range labels {
				names = append(names, l.GetName())
			}
			assert.ElementsMatch(t, []string{"query_name", "operation", "status"}, names)
			// SQL 本文や bind 値がラベル値に混入していないこと。
			assert.Equal(t, "select", labelValue(duration.GetMetric()[0], "operation"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗時はerror counterをerror_class付きで増やす", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			recorder := NewQueryRecorder(reg)

			recorder.Observe(context.Background(), driver.QueryAttrs{
				QueryName:  "user.create",
				Operation:  "insert",
				Status:     "error",
				ErrorClass: "constraint",
				Duration:   2 * time.Millisecond,
			})

			errs := findMetricFamily(t, reg, "rdb_query_errors_total")
			require.NotNil(t, errs)
			require.Len(t, errs.GetMetric(), 1)

			m := errs.GetMetric()[0]
			assert.InDelta(t, 1.0, m.GetCounter().GetValue(), 0)
			assert.Equal(t, "constraint", labelValue(m, "error_class"))
			assert.Equal(t, "insert", labelValue(m, "operation"))
			assert.Equal(t, "user.create", labelValue(m, "query_name"))
		})
	})
}

func TestRegisterOrExisting(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一レジストリに二重生成しても同じメトリクスへ記録できる", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			first := NewQueryRecorder(reg)
			// 二度目の生成でも AlreadyRegistered を吸収し、既存コレクタを再利用する。
			second := NewQueryRecorder(reg)

			attrs := driver.QueryAttrs{
				QueryName: "user.find_by_id",
				Operation: "select",
				Status:    "success",
				Duration:  time.Millisecond,
			}
			first.Observe(context.Background(), attrs)
			second.Observe(context.Background(), attrs)

			duration := findMetricFamily(t, reg, "rdb_query_duration_seconds")
			require.NotNil(t, duration)
			require.Len(t, duration.GetMetric(), 1)
			// 別インスタンス経由でも同一メトリクスに 2 回記録される。
			assert.Equal(t, uint64(2), duration.GetMetric()[0].GetHistogram().GetSampleCount())
		})
	})
}
