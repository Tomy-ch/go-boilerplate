//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
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

// NewBuildInfo は、ビルド時 ldflags で埋め込まれた Version / Revision / BuildDate パッケージ変数から BuildInfo を生成して返します。
func NewBuildInfo() BuildInfo {
	return &buildInfo{
		version:   Version,
		revision:  Revision,
		buildDate: BuildDate,
	}
}

// Version はバージョン文字列を返します。
func (bi *buildInfo) Version() string { return bi.version }

// Revision はリビジョン文字列を返します。
func (bi *buildInfo) Revision() string { return bi.revision }

// BuildDate はビルド日時文字列を返します。
func (bi *buildInfo) BuildDate() string { return bi.buildDate }
