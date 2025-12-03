package observability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getCallerFullName(t *testing.T) {
	t.Parallel()

	got := getCallerFullName()
	require.NotEmpty(t, got)
}
