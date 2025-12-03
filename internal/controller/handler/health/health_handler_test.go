package health

import (
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/handlertest/testassert"
	"boilerplate-go/internal/controller/handler/handlertest/testinstance"
	"boilerplate-go/internal/controller/handler/health/gen"

	"github.com/stretchr/testify/require"
)

const targetPath = "/health"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e, _, tf, _ := testinstance.NewTestInstanceForBindHandler(t)
	BindHandler(e, tf)

	expectedMethods := []string{http.MethodGet}
	testassert.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	testassert.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func TestGetHealth(t *testing.T) {
	t.Parallel()

	ctx, _, _, lt := testinstance.NewTestInstancesForImplementedUsecase(t)
	s := &server{
		tracer: lt,
	}

	expectedResponse := gen.ResponseHealth{Status: "ok"}

	resp, err := s.GetHealth(ctx, gen.GetHealthRequestObject{})
	require.NoError(t, err)

	actual, ok := resp.(gen.GetHealth200JSONResponse)
	require.True(t, ok)

	require.Equal(t, expectedResponse, gen.ResponseHealth(actual))
}
