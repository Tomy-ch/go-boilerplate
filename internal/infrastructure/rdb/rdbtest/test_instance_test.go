package rdbtest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTestInstances(t *testing.T) {
	t.Parallel()
	ctx, db, txm, logger, location := NewTestInstances(t)
	require.NotNil(t, ctx)
	require.NotNil(t, db)
	require.NotNil(t, txm)
	require.NotNil(t, logger)
	require.NotNil(t, location)
}
