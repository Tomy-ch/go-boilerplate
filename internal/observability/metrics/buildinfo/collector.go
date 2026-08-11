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
func NewCollector(appCfg *config.ApplicationConfig, bi system.BuildInfo) *Collector {
	// ラベル名と値を 1 箇所で対にし、位置依存の取り違え（version に revision 値が
	// 入る等のサイレントな順序ずれ）を構造的に排除します。
	pairs := []struct{ name, value string }{
		{labelService, normalize(appCfg.Name())},
		{labelEnvironment, normalize(appCfg.Env())},
		{labelVersion, normalize(bi.Version())},
		{labelRevision, normalize(bi.Revision())},
		{labelBuildDate, normalize(bi.BuildDate())},
		{labelGoVersion, normalize(runtime.Version())},
	}
	names := make([]string, len(pairs))
	values := make([]string, len(pairs))
	for i, p := range pairs {
		names[i] = p.name
		values[i] = p.value
	}
	return &Collector{
		desc:        prometheus.NewDesc(metricName, metricHelp, names, nil),
		labelValues: values,
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
func Register(c *Collector) error {
	return register(prometheus.DefaultRegisterer, c)
}

// register は、指定された Registerer に Collector を登録します。
//
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
