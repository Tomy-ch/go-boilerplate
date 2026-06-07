package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBuildInfo(t *testing.T) {
	t.Parallel()

	expected := &buildInfo{
		version:   "dev",
		revision:  "none",
		buildDate: "2024-12-31T21:00:00Z",
	}

	bi := NewBuildInfo()

	assert.Equal(t, expected, bi)
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

	assert.Equal(t, expectedVersion, actual.Version())
	assert.Equal(t, expectedRevision, actual.Revision())
	assert.Equal(t, expectedBuildDate, actual.BuildDate())
}
