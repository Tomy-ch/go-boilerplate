package integration

import (
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/handlertest"
	"boilerplate-go/internal/controller/handler/health"
	"boilerplate-go/internal/controller/handler/health/gen"
)

func TestHealth_Integration(t *testing.T) {
	t.Parallel()

	t.Run("GET /healthのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e, _, tf, _ := handlertest.NewTestInstanceForBindHandler(t)

		health.BindHandler(e, tf)

		actual := StartServer(t, e).DoJSON(http.MethodGet, "/health", nil, nil)
		AssertJSONResponse(t, gen.ResponseHealth{}, actual)
	})
}
