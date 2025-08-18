package binder

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	e := echo.New()
	expectedType := &echo.DefaultBinder{}
	New(e)
	require.NotNil(t, e.Binder)
	require.IsType(t, expectedType, e.Binder)
}

func TestNewBinder(t *testing.T) {
	t.Parallel()
	expectedType := &echo.DefaultBinder{}
	actual := NewBinder()
	require.NotNil(t, actual)
	require.IsType(t, expectedType, actual)
}
