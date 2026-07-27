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
			// operation ラベルには分類済みの固定 enum（例: select）が入る。
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

			// エラー時でも duration は status=error 付きで 1 件記録される。
			duration := findMetricFamily(t, reg, "rdb_query_duration_seconds")
			require.NotNil(t, duration)
			require.Len(t, duration.GetMetric(), 1)
			dm := duration.GetMetric()[0]
			assert.Equal(t, uint64(1), dm.GetHistogram().GetSampleCount())
			assert.Equal(t, "error", labelValue(dm, "status"))
		})
	})
}

func Test_registerOrExisting(t *testing.T) {
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

		t.Run("未登録のコレクタは生成したものをそのまま返す", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			counter := prometheus.NewCounterVec(
				prometheus.CounterOpts{Name: "register_or_existing_fresh", Help: "h"}, []string{"l"})

			got := registerOrExisting(reg, counter)
			assert.Same(t, counter, got)
		})

		t.Run("同名で型一致の既存コレクタがあれば既存を返す", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			opts := prometheus.CounterOpts{Name: "register_or_existing_same_type", Help: "h"}

			first := prometheus.NewCounterVec(opts, []string{"l"})
			require.Same(t, first, registerOrExisting(reg, first))

			second := prometheus.NewCounterVec(opts, []string{"l"})
			// 2 度目は AlreadyRegistered を吸収し、既存(first)を返す。
			assert.Same(t, first, registerOrExisting(reg, second))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同名だが既存コレクタの型が一致しなければpanicする", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			const name = "register_or_existing_type_mismatch"

			counter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: "h"}, []string{"l"})
			require.NoError(t, reg.Register(counter))

			// 同名・同 descriptor だが型が *HistogramVec のため型アサーションが失敗し panic する。
			hist := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name, Help: "h"}, []string{"l"})
			assert.Panics(t, func() { registerOrExisting(reg, hist) })
		})

		t.Run("AlreadyRegistered以外の登録失敗はpanicする", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			const name = "register_or_existing_dim_conflict"

			// 同名だが help 文字列が異なるため dimHash が衝突し、AlreadyRegistered ではない登録エラーになる。
			require.NoError(t, reg.Register(
				prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: "h1"}, []string{"l"})))

			conflict := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: "h2"}, []string{"l"})
			assert.Panics(t, func() { registerOrExisting(reg, conflict) })
		})
	})
}

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
