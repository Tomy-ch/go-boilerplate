package ops

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsOpsPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "metrics path",
			path: "/metrics",
			want: true,
		},
		{
			name: "health path",
			path: "/health",
			want: true,
		},
		{
			name: "healthz path",
			path: "/healthz",
			want: true,
		},
		{
			name: "ready path",
			path: "/ready",
			want: true,
		},
		{
			name: "version path",
			path: "/version",
			want: true,
		},
		{
			name: "non-ops path",
			path: "/non-ops",
			want: false,
		},
		{
			name: "root path",
			path: "/",
			want: false,
		},
		{
			name: "ops path with trailing slash",
			path: "/metrics/",
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsOpsPath(tt.path)
			require.Equal(t, tt.want, got)
		})
	}
}
