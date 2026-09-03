package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"go-boilerplate/internal/controller/handler/ready"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/healthcheck"
	mock_healthcheck "go-boilerplate/internal/usecase/healthcheck/mock"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestReady_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /readyがUsecaseのDTOを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_healthcheck.NewMockUsecase(ctrl)
			mockApp.EXPECT().CheckHealth(gomock.Any()).Return(&healthcheck.DTO{}, nil)

			ready.BindHandler(e, tf, mockApp)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/ready", nil, nil)
			AssertJSONResponseType[healthcheck.DTO](t, actual)
		})

		t.Run("縮退した依存があってもGET /readyは200で状態を並べる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_healthcheck.NewMockUsecase(ctrl)
			mockApp.EXPECT().CheckHealth(gomock.Any()).Return(&healthcheck.DTO{
				Status: healthcheck.Degraded,
				Dependencies: []healthcheck.DependencyStatus{
					{Name: "realtime", Status: healthcheck.Degraded},
				},
			}, nil)

			ready.BindHandler(e, tf, mockApp)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/ready", nil, nil)

			// 200 であることが要点。ここが 503 になると、Realtime だけの不調で instance が
			// load balancer から外れ、通常の HTTP まで止まる（親受入基準 25）。
			require.Equal(t, http.StatusOK, actual.StatusCode)

			resBody, err := io.ReadAll(actual.Body)
			require.NoError(t, err)

			var body struct {
				Status       string `json:"status"`
				Dependencies []struct {
					Name   string `json:"name"`
					Status string `json:"status"`
				} `json:"dependencies"`
			}
			require.NoError(t, json.Unmarshal(resBody, &body))
			assert.Equal(t, "degraded", body.Status)
			require.Len(t, body.Dependencies, 1)
			assert.Equal(t, "realtime", body.Dependencies[0].Name)
			assert.Equal(t, "degraded", body.Dependencies[0].Status)
		})
	})
}
