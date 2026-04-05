package skipper

import (
	"context"
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
		{"health", "/health", true},
		{"other path", "/v1/users", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctx := context.Background()

			req := httptest.NewRequestWithContext(ctx, http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			got := sk(c)
			require.Equal(t, tc.want, got)
		})
	}
}
