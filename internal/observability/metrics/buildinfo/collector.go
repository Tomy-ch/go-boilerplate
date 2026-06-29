// Package buildinfo は、アプリケーションのビルド・バージョン・ランタイム情報を
// Prometheus の info gauge (app_build_info) として公開する機能を提供します。
//
// 値は常に 1 であり、意味はラベル (service / environment / version / revision /
// build_date / go_version) に持たせます。これらの値はプロセス起動後に変化しない
// ため、ラベル値は DI 結線時に一度だけ解決・正規化して保持し、スクレイプごとの
// 計算は行いません。
package buildinfo

import (
	"runtime"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/system"
	"go-boilerplate/pkg/xerrors"

	"github.com/prometheus/client_golang/prometheus"
)

// Collector は、ビルド情報を info gauge として公開する prometheus.Collector です。
type Collector struct {
	desc        *prometheus.Desc
	labelValues []string
}

// NewCollector は、ラベル値を結線時に一度だけ解決・正規化して Collector を生成します。
//
// version / revision / build_date は system.BuildInfo、go_version は runtime.Version()、
// service / environment は config.ApplicationConfig から取得します。
// いずれも /version と同一の source を用います。
func NewCollector(appCfg *config.ApplicationConfig, bi system.BuildInfo) *Collector {
	return &Collector{
		desc: prometheus.NewDesc(
			metricName,
			metricHelp,
			[]string{
				labelService,
				labelEnvironment,
				labelVersion,
				labelRevision,
				labelBuildDate,
				labelGoVersion,
			},
			nil,
		),
		labelValues: []string{
			normalize(appCfg.Name()),
			normalize(appCfg.Env()),
			normalize(bi.Version()),
			normalize(bi.Revision()),
			normalize(bi.BuildDate()),
			normalize(runtime.Version()),
		},
	}
}

// Describe は、Collector のメトリクスの説明を Prometheus に提供します。
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect は、保持済みのラベル値で値 1 の info gauge を emit します。
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1, c.labelValues...)
}

// Register は、Collector を Prometheus のデフォルトレジストリに登録します。
//
// 重複登録は AlreadyRegisteredError を無視して安全にスキップします。
func Register(c *Collector) error {
	return register(prometheus.DefaultRegisterer, c)
}

// register は、指定された Registerer に Collector を登録します。
//
// グローバルな DefaultRegisterer への依存を Register から切り離し、テストが任意の
// レジストリを注入して並列実行できるようにするための内部ヘルパーです。
// 重複登録は AlreadyRegisteredError を無視して安全にスキップします。
func register(reg prometheus.Registerer, c *Collector) error {
	err := reg.Register(c)
	if err != nil {
		var alreadyRegisteredErr prometheus.AlreadyRegisteredError
		if xerrors.As(err, &alreadyRegisteredErr) {
			return nil
		}
		return err
	}
	return nil
}
