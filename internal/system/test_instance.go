package system

import "testing"

func NewTestBuildInfo(t *testing.T, version, revision, buildDate string) BuildInfo {
	t.Helper()
	return &buildInfo{
		version:   version,
		revision:  revision,
		buildDate: buildDate,
	}
}
