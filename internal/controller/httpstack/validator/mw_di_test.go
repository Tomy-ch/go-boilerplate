package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestCoreModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, CoreModule())
}
