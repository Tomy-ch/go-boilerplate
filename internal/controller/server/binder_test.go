package server

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestNewBinder(t *testing.T) {
	expectedType := &echo.DefaultBinder{}
	actual := NewBinder()
	require.NotNil(t, actual)
	require.IsType(t, expectedType, actual)
}
