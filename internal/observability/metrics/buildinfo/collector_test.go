package buildinfo

import (
	"errors"
	"runtime"
	"testing"

	"go-boilerplate/internal/config"
	mock_system "go-boilerplate/internal/system/mock"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

// errRegisterFailed は、fakeRegisterer が Register で返すテスト用の番兵エラーです。
var errRegisterFailed = errors.New("register failed")

// fakeRegisterer は、Register が常に errRegisterFailed を返す prometheus.Registerer の
// テスト用スタブです。AlreadyRegisteredError 以外のエラー分岐を到達させるために用います。
type fakeRegisterer struct{}

// gatherLabels は、Collector を専用レジストリへ登録して app_build_info の
// ラベルと値を収集するテスト補助関数です。
func gatherLabels(t *testing.T, c *Collector) (map[string]string, float64) {
	t.Helper()

	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(c))

	families, err := reg.Gather()
	require.NoError(t, err)

	var target *dto.MetricFamily
	for _, mf := range families {
		if mf.GetName() == metricName {
			target = mf
			break
		}
	}
	require.NotNil(t, target)
	assert.Equal(t, dto.MetricType_GAUGE, target.GetType())
	require.Len(t, target.GetMetric(), 1)

	metric := target.GetMetric()[0]
	labels := make(map[string]string, len(metric.GetLabel()))
	for _, lp := range metric.GetLabel() {
		labels[lp.GetName()] = lp.GetValue()
	}
	return labels, metric.GetGauge().GetValue()
}

func (fakeRegisterer) Register(prometheus.Collector) error  { return errRegisterFailed }
func (fakeRegisterer) MustRegister(...prometheus.Collector) {}
func (fakeRegisterer) Unregister(prometheus.Collector) bool { return false }

func TestNewCollector(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Collectorが生成される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.5.0")
			bi.EXPECT().Revision().Return("abcdef1")
			bi.EXPECT().BuildDate().Return("2026-06-28T17:00:00Z")

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))

			c := NewCollector(appCfg, bi)
			require.NotNil(t, c)
		})
	})
}

func TestCollector_Describe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Descが1つだけ送られメトリクス名を含む", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.5.0")
			bi.EXPECT().Revision().Return("abcdef1")
			bi.EXPECT().BuildDate().Return("2026-06-28T17:00:00Z")

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))

			ch := make(chan *prometheus.Desc, 1)
			NewCollector(appCfg, bi).Describe(ch)
			close(ch)

			var descs []*prometheus.Desc
			for d := range ch {
				descs = append(descs, d)
			}

			require.Len(t, descs, 1)
			assert.Contains(t, descs[0].String(), metricName)
		})
	})
}

func TestCollector_Collect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全てのビルド情報がラベルに反映され値が1になる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.5.0")
			bi.EXPECT().Revision().Return("abcdef1")
			bi.EXPECT().BuildDate().Return("2026-06-28T17:00:00Z")

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))

			labels, value := gatherLabels(t, NewCollector(appCfg, bi))

			assert.Equal(t, 1.0, value)
			assert.Equal(t, "TestApp", labels[labelService])
			assert.Equal(t, "test", labels[labelEnvironment])
			assert.Equal(t, "v1.5.0", labels[labelVersion])
			assert.Equal(t, "abcdef1", labels[labelRevision])
			assert.Equal(t, "2026-06-28T17:00:00Z", labels[labelBuildDate])
			assert.Equal(t, runtime.Version(), labels[labelGoVersion])
		})

		t.Run("ビルド情報が空の場合はunknownに丸められる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("")
			bi.EXPECT().Revision().Return("")
			bi.EXPECT().BuildDate().Return("")

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			appCfg.SetApplicationEnv(t, "")

			labels, value := gatherLabels(t, NewCollector(appCfg, bi))

			assert.Equal(t, 1.0, value)
			assert.Equal(t, unknownValue, labels[labelEnvironment])
			assert.Equal(t, unknownValue, labels[labelVersion])
			assert.Equal(t, unknownValue, labels[labelRevision])
			assert.Equal(t, unknownValue, labels[labelBuildDate])
			// go_version は runtime.Version() のため常に非空であり unknown にならない。
			assert.NotEqual(t, unknownValue, labels[labelGoVersion])
		})

		t.Run("禁止ラベルが含まれない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.5.0")
			bi.EXPECT().Revision().Return("abcdef1")
			bi.EXPECT().BuildDate().Return("2026-06-28T17:00:00Z")

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))

			labels, _ := gatherLabels(t, NewCollector(appCfg, bi))

			forbidden := []string{
				"hostname", "pod_name", "container_id", "instance_id",
				"git_branch", "build_url", "image_digest", "full_image",
				"token", "registry", "commit",
			}
			for _, key := range forbidden {
				_, ok := labels[key]
				assert.False(t, ok, "禁止ラベル %q が含まれてはならない", key)
			}
			assert.Len(t, labels, 6)
		})
	})
}

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回登録は成功する", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.5.0")
			bi.EXPECT().Revision().Return("abcdef1")
			bi.EXPECT().BuildDate().Return("2026-06-28T17:00:00Z")

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))

			require.NoError(t, register(reg, NewCollector(appCfg, bi)))
		})

		t.Run("デフォルトレジストリに登録できる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.5.0").AnyTimes()
			bi.EXPECT().Revision().Return("abcdef1").AnyTimes()
			bi.EXPECT().BuildDate().Return("2026-06-28T17:00:00Z").AnyTimes()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))

			require.NoError(t, Register(NewCollector(appCfg, bi)))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複登録は無視される", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.5.0").AnyTimes()
			bi.EXPECT().Revision().Return("abcdef1").AnyTimes()
			bi.EXPECT().BuildDate().Return("2026-06-28T17:00:00Z").AnyTimes()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			c := NewCollector(appCfg, bi)

			require.NoError(t, register(reg, c))
			require.NoError(t, register(reg, c))
		})

		t.Run("AlreadyRegisteredError以外のエラーはそのまま返される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.5.0").AnyTimes()
			bi.EXPECT().Revision().Return("abcdef1").AnyTimes()
			bi.EXPECT().BuildDate().Return("2026-06-28T17:00:00Z").AnyTimes()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))

			err := register(fakeRegisterer{}, NewCollector(appCfg, bi))
			require.ErrorIs(t, err, errRegisterFailed)
		})
	})
}
