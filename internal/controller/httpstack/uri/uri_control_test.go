package uri

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Middleware())
}
