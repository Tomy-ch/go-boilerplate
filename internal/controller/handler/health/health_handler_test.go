package health

import (
	"context"
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/handlertest"
	"boilerplate-go/internal/controller/handler/health/gen"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

const targetPath = "/health"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	BindHandler(e)

	expectedMethods := []string{http.MethodGet}
	handlertest.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	handlertest.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func TestGetHealth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expectedResponse := gen.ResponseHealth{Status: "ok"}

	s := &server{}
	resp, err := s.GetHealth(ctx, gen.GetHealthRequestObject{})
	require.NoError(t, err)

	actual, ok := resp.(gen.GetHealth200JSONResponse)
	require.True(t, ok)

	require.Equal(t, expectedResponse, gen.ResponseHealth(actual))
}
