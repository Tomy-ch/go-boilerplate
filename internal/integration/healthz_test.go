package integration

import (
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/handlertest/testinstance"
	"boilerplate-go/internal/controller/handler/health/gen"
	"boilerplate-go/internal/controller/handler/healthz"
)

func TestHealthz_Integration(t *testing.T) {
	t.Parallel()

	t.Run("GET /healthzのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e, _, tf, _ := testinstance.NewTestInstanceForBindHandler(t)

		healthz.BindHandler(e, tf)

		actual := StartServer(t, e).DoJSON(http.MethodGet, "/healthz", nil, nil)
		AssertJSONResponse(t, gen.ResponseHealth{}, actual)
	})
}
