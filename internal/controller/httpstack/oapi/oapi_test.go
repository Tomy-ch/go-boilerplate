package oapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/controller/ctxhelper"
	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	spec := &openapi3.T{}
	mw := Middleware(spec, nil, nil)

	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	handler := mw(func(_ echo.Context) error {
		return nil
	})

	_ = handler(c)

	// ミドルウェアが Authn スロットを仕込んでいることを確認（Set が成功し Get で読める）。
	a, err := authbd.New("u1", authbd.ProviderMock, nil, nil)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(c.Request().Context(), *a))

	got, ok := ctxhelper.GetAuthn(c.Request().Context())
	assert.True(t, ok)
	assert.Equal(t, a.Subject(), got.Subject())
}
