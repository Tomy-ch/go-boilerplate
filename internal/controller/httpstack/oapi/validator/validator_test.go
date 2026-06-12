package validator

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestGetValidator(t *testing.T) {
	t.Parallel()

	validator, err := GetValidator()
	require.NoError(t, err)
	require.IsType(t, &openapi3.T{}, validator)
}
