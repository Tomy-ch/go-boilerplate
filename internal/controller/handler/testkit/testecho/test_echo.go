// Package testecho は、Echoフレームワークを使用したハンドラーのテストユーティリティを提供します。
package testecho

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/errorhandler"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

var (
	errTargetUnset  = errors.New("RequestURL か RoutePattern のいずれかを設定してください")
	errModeConflict = errors.New("RequestURL は RoutePattern/PathParams と併用できません")
)

type EchoTestParam struct {
	Name  string
	Value string
}

type EchoTestClient struct {
	t            *testing.T
	e            *echo.Echo
	method       string
	routePattern string
	requestURL   string
	body         io.Reader
	headers      http.Header
	pathParams   []EchoTestParam
	queryParams  []EchoTestParam
}

// NewEchoTestClient はテスト用のEchoクライアントを生成します。
func NewEchoTestClient(t *testing.T, e *echo.Echo) *EchoTestClient {
	t.Helper()
	return &EchoTestClient{
		t:       t,
		e:       e,
		headers: make(http.Header),
	}
}

// WithAppErrorHandler は、本番相当のエラーハンドラを Echo に設定します。
// 渡された Echo の HTTPErrorHandler を上書きする副作用を持ちます。
func (c *EchoTestClient) WithAppErrorHandler() *EchoTestClient {
	c.t.Helper()
	cfg := config.MockConfigForTest(c.t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(c.t)
	errorhandler.New(c.e, logging.NewTestLogger(c.t), lf, obsCfg)
	return c
}

// Method はHTTPメソッドを設定します。
func (c *EchoTestClient) Method(m string) *EchoTestClient {
	c.t.Helper()
	c.method = m
	return c
}

// RoutePattern はルートパターンを設定します。
//
// 例: /users/:id, /products/:id
func (c *EchoTestClient) RoutePattern(p string) *EchoTestClient {
	c.t.Helper()
	c.routePattern = p
	return c
}

// RequestURL は実際に叩くURLを設定します。
// (パスパラメータやクエリパラメータを含めて指定することができます。)
//
// 例: /users/123?limit=10, /products/456?sort=asc
func (c *EchoTestClient) RequestURL(u string) *EchoTestClient {
	c.t.Helper()
	c.requestURL = u
	return c
}

// JSONBody はJSON形式のリクエストボディを設定します。
func (c *EchoTestClient) JSONBody(v any) *EchoTestClient {
	c.t.Helper()
	data, err := json.Marshal(v)
	require.NoError(c.t, err)
	c.body = bytes.NewReader(data)
	c.Header(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return c
}

// RawBody は生のリクエストボディを設定します。
//
// JSON形式は、 JSONBody() を使用してください。
func (c *EchoTestClient) RawBody(r io.Reader, contentType string) *EchoTestClient {
	c.t.Helper()
	c.body = r
	if contentType != "" {
		c.Header(echo.HeaderContentType, contentType)
	}
	return c
}

// Header はリクエストヘッダーを設定します。
func (c *EchoTestClient) Header(k, v string) *EchoTestClient {
	c.t.Helper()
	c.headers.Set(k, v)
	return c
}

// AuthBearer はBearerトークンを設定します。
func (c *EchoTestClient) AuthBearer(token string) *EchoTestClient {
	c.t.Helper()
	return c.Header(echo.HeaderAuthorization, "Bearer "+token)
}

// PathParams はパスパラメータを設定します。
func (c *EchoTestClient) PathParams(params []EchoTestParam) *EchoTestClient {
	c.t.Helper()
	c.pathParams = params
	return c
}

// QueryParams はクエリパラメータを設定します。
func (c *EchoTestClient) QueryParams(params []EchoTestParam) *EchoTestClient {
	c.t.Helper()
	c.queryParams = params
	return c
}

// Build はテスト用のHTTPリクエストとレスポンスレコーダー、echo.Contextを構築します。
//
// requestURL モードではルータ解決により echo.Context のパスが設定されます。
func (c *EchoTestClient) Build() (*http.Request, *httptest.ResponseRecorder, echo.Context) {
	c.t.Helper()

	req, rec := c.buildRequest()
	ec := c.e.NewContext(req, rec)

	if c.requestURL != "" {
		c.e.Router().Find(c.method, req.URL.Path, ec)
		return req, rec, ec
	}

	ec.SetPath(c.routePattern)
	if len(c.pathParams) > 0 {
		names := make([]string, len(c.pathParams))
		values := make([]string, len(c.pathParams))
		for i, p := range c.pathParams {
			names[i], values[i] = p.Name, p.Value
		}
		ec.SetParamNames(names...)
		ec.SetParamValues(values...)
	}

	return req, rec, ec
}

// Serve は、結合テスト用に起動したEchoインスタンスに対してリクエストを送信し、レスポンスを取得します。
func (c *EchoTestClient) Serve() *httptest.ResponseRecorder {
	c.t.Helper()
	req, rec := c.buildRequest()
	c.e.ServeHTTP(rec, req)
	return rec
}

// resolveTarget はリクエスト先URLを決定し、モードの排他違反を検出します。
//
// requestURL モード(ルータ登録済みの Echo 前提)と routePattern/pathParams モードは排他です。
func (c *EchoTestClient) resolveTarget() (string, error) {
	switch {
	case c.requestURL != "":
		if c.routePattern != "" || len(c.pathParams) > 0 {
			return "", errModeConflict
		}
		return c.requestURL, nil
	case c.routePattern != "":
		return c.routePattern, nil
	default:
		return "", errTargetUnset
	}
}

// buildRequest はリクエストとレスポンスレコーダーを構築します。
func (c *EchoTestClient) buildRequest() (*http.Request, *httptest.ResponseRecorder) {
	c.t.Helper()

	target, err := c.resolveTarget()
	if err != nil {
		c.t.Fatal(err)
	}

	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, c.method, target, c.body)
	for k, vv := range c.headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	q := req.URL.Query()
	for _, p := range c.queryParams {
		q.Add(p.Name, p.Value)
	}
	req.URL.RawQuery = q.Encode()

	return req, httptest.NewRecorder()
}
