package security

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCookieModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, CookieModule())
}
