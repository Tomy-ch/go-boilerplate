package validator

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	validator, err := GetValidator()
	require.NoError(t, err)

	require.NotNil(t, Middleware(validator))
}

func TestGetValidator(t *testing.T) {
	t.Parallel()

	validator, err := GetValidator()
	require.NoError(t, err)
	require.IsType(t, &openapi3.T{}, validator)
}

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestCoreModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, CoreModule())
}
