package inbound

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, OpenAPIModule())
}

func TestOpenAPIUseMiddleware(t *testing.T) {
	t.Parallel()

	noopFn := func(context.Context, *openapi3filter.AuthenticationInput) error { return nil }

	mw := OpenAPIMiddleware(&openapi3.T{}, nil, noopFn)
	require.Equal(t, validatorUsePriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
