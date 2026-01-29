package skipper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestNewSkipper(t *testing.T) {
	t.Parallel()

	sk := New()

	cases := []struct {
		name string
		path string
		want bool
	}{
		{"metrics", "/metrics", true},
		{"health", "/health", true},
		{"healthz", "/healthz", true},
		{"ready", "/ready", true},
		{"version", "/version", true},
		{"other path", "/api/v1/users", false},
		{"root", "/", false},
		{"similar but not exact", "/healthcheck", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			got := sk(c)
			require.Equal(t, tc.want, got)
		})
	}
}
