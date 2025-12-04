package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBuildInfo(t *testing.T) {
	t.Parallel()

	expected := &buildInfo{
		version:   "dev",
		revision:  "none",
		buildDate: "2024-12-31T21:00:00Z",
	}

	bi := NewBuildInfo()

	require.Equal(t, expected, bi)
}

func TestBuildInfoMethods(t *testing.T) {
	t.Parallel()

	expectedVersion := "1.0.0"
	expectedRevision := "abc123"
	expectedBuildDate := "2024-12-31T21:00:00Z"

	actual := &buildInfo{
		version:   expectedVersion,
		revision:  expectedRevision,
		buildDate: expectedBuildDate,
	}

	require.Equal(t, expectedVersion, actual.Version())
	require.Equal(t, expectedRevision, actual.Revision())
	require.Equal(t, expectedBuildDate, actual.BuildDate())
}
