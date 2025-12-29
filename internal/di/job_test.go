package di

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewJobCore(t *testing.T) {
	t.Parallel()

	jobApp := NewJobCore()
	require.NotNil(t, jobApp)
}

func TestRunJob(t *testing.T) {
	t.Parallel()

	start, stop := RunJob()
	require.NotNil(t, start)
	require.NotNil(t, stop)
}
