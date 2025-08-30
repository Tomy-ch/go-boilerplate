package health

import (
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

	e := echo.New()
	BindHandler(e)

	_, res, ctx := handlertest.
		NewEchoTestClient(t, e).
		Method(http.MethodGet).
		RequestURL(targetPath).
		Build()

	s := &server{}
	err := s.GetHealth(ctx)
	require.NoError(t, err)

	expectedResponse := gen.ResponseHealth{Status: "ok"}

	handlertest.AssertJSONEqual(t, http.StatusOK, expectedResponse, res)
}
