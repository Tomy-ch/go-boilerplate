//go:generate mockgen -source=$GOFILE -destination=mock/mock_buildinfo.gen.go -package=mock_$GOPACKAGE
package system

// BuildInfo は、アプリケーションのビルド情報を提供するインターフェースです。
type BuildInfo interface {
	Version() string
	Revision() string
	BuildDate() string
}

// buildInfo は、アプリケーションのビルド情報を表します。
type buildInfo struct {
	version   string
	revision  string
	buildDate string
}

// NewBuildInfo は、BuildInfo の新しいインスタンスを作成します。
func NewBuildInfo() BuildInfo {
	return &buildInfo{
		version:   Version,
		revision:  Revision,
		buildDate: BuildDate,
	}
}

// Version は、buildInfo からバージョン情報を取得します。
func (bi *buildInfo) Version() string { return bi.version }

// Revision は、buildInfo からリビジョン情報を取得します。
func (bi *buildInfo) Revision() string { return bi.revision }

// BuildDate は、buildInfo からビルド日時情報を取得します。
func (bi *buildInfo) BuildDate() string { return bi.buildDate }
