package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServeCommand(t *testing.T) {
	t.Parallel()

	cmd := NewServeCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "serve", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}
