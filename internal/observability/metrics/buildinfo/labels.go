package buildinfo

const (
	// metricName は、ビルド情報を公開する info gauge のメトリクス名です。
	metricName = "app_build_info"
	// metricHelp は、メトリクスの説明文です。値は常に 1 です。
	metricHelp = "Application build and runtime information. The value is always 1."

	labelService     = "service"
	labelEnvironment = "environment"
	labelVersion     = "version"
	labelRevision    = "revision"
	labelBuildDate   = "build_date"
	labelGoVersion   = "go_version"

	// unknownValue は、ラベル値が空の場合に丸める既定値です。
	unknownValue = "unknown"
)

// normalize は、空文字のラベル値を unknownValue に丸めます。
func normalize(v string) string {
	if v == "" {
		return unknownValue
	}
	return v
}
