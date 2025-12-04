package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTestBuildInfo(t *testing.T) {
	t.Parallel()

	version := "1.0.0"
	revision := "abc123"
	buildDate := "2024-06-01T12:00:00Z"

	expected := &buildInfo{
		version:   version,
		revision:  revision,
		buildDate: buildDate,
	}

	actual := NewTestBuildInfo(t, version, revision, buildDate)
	require.Equal(t, expected, actual)
}
