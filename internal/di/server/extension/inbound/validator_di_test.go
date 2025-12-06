package inbound

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestValidatorModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, ValidatorModule())
}

func TestValidatorUseMiddleware(t *testing.T) {
	t.Parallel()

	mw := ValidatorMiddleware(&openapi3.T{})
	require.Equal(t, validatorUsePriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
