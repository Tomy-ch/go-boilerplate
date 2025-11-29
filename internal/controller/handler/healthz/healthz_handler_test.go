package healthz

import (
	"context"
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/handlertest"
	"boilerplate-go/internal/controller/handler/healthz/gen"

	"github.com/stretchr/testify/require"
)

const targetPath = "/healthz"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e, _ := handlertest.NewBindHandlerTestInstance(t)
	BindHandler(e)

	expectedMethods := []string{http.MethodGet}

	handlertest.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	handlertest.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func TestGetHealthz(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := &server{}
	expectedResponse := gen.ResponseHealth{Status: "ok"}

	resp, err := s.GetHealthz(ctx, gen.GetHealthzRequestObject{})
	require.NoError(t, err)

	actual, ok := resp.(gen.GetHealthz200JSONResponse)
	require.True(t, ok)

	require.Equal(t, expectedResponse, gen.ResponseHealth(actual))
}
