// Package testecho は、Echoフレームワークを使用したハンドラーのテストユーティリティを提供します。
package testecho

import (
	"bytes"
	"context"
	"encoding/json"
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
	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	errorhandler.New(e, logging.NewTestLogger(t), lf, obsCfg)
	return &EchoTestClient{
		t:       t,
		e:       e,
		headers: make(http.Header),
	}
}

// Method はHTTPメソッドを設定します。
func (b *EchoTestClient) Method(m string) *EchoTestClient {
	b.t.Helper()
	b.method = m
	return b
}

// RoutePattern はルートパターンを設定します。
//
// 例: /users/:id, /products/:id
func (b *EchoTestClient) RoutePattern(p string) *EchoTestClient {
	b.t.Helper()
	b.routePattern = p
	return b
}

// RequestURL は実際に叩くURLを設定します。
// (パスパラメータやクエリパラメータを含めて指定することができます。)
//
// 例: /users/123?limit=10, /products/456?sort=asc
func (b *EchoTestClient) RequestURL(u string) *EchoTestClient {
	b.t.Helper()
	b.requestURL = u
	return b
}

// JSONBody はJSON形式のリクエストボディを設定します。
func (b *EchoTestClient) JSONBody(v any) *EchoTestClient {
	b.t.Helper()
	data, err := json.Marshal(v)
	require.NoError(b.t, err)
	b.body = bytes.NewReader(data)
	b.Header(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return b
}

// RawBody は生のリクエストボディを設定します。
//
// JSON形式は、 JSONBody() を使用してください。
func (b *EchoTestClient) RawBody(r io.Reader, contentType string) *EchoTestClient {
	b.t.Helper()
	b.body = r
	if contentType != "" {
		b.Header(echo.HeaderContentType, contentType)
	}
	return b
}

// Header はリクエストヘッダーを設定します。
func (b *EchoTestClient) Header(k, v string) *EchoTestClient {
	b.t.Helper()
	b.headers.Set(k, v)
	return b
}

// AuthBearer はBearerトークンを設定します。
func (b *EchoTestClient) AuthBearer(token string) *EchoTestClient {
	b.t.Helper()
	return b.Header(echo.HeaderAuthorization, "Bearer "+token)
}

// PathParams はパスパラメータを設定します。
func (b *EchoTestClient) PathParams(params []EchoTestParam) *EchoTestClient {
	b.t.Helper()
	b.pathParams = params
	return b
}

// QueryParams はクエリパラメータを設定します。
func (b *EchoTestClient) QueryParams(params []EchoTestParam) *EchoTestClient {
	b.t.Helper()
	b.queryParams = params
	return b
}

// Build はテスト用のHTTPリクエストとレスポンスレコーダーを構築し、EchoTestClientの終端となります。
func (b *EchoTestClient) Build() (*http.Request, *httptest.ResponseRecorder, echo.Context) {
	b.t.Helper()

	var target string
	switch {
	case b.requestURL != "":
		target = b.requestURL
	case b.routePattern != "":
		target = b.routePattern
	default:
		b.t.Fatal("requestURL か routePattern のいずれかを設定してください")
	}

	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, b.method, target, b.body)
	for k, vv := range b.headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	c := b.e.NewContext(req, rec)

	q := req.URL.Query()
	for _, p := range b.queryParams {
		q.Add(p.Name, p.Value)
	}
	req.URL.RawQuery = q.Encode()

	if b.requestURL != "" {
		b.e.Router().Find(b.method, req.URL.Path, c)
	} else {
		c.SetPath(b.routePattern)
		if len(b.pathParams) > 0 {
			names := make([]string, len(b.pathParams))
			values := make([]string, len(b.pathParams))
			for i, p := range b.pathParams {
				names[i], values[i] = p.Name, p.Value
			}
			c.SetParamNames(names...)
			c.SetParamValues(values...)
		}
	}

	return req, rec, c
}

// Serve は、結合テスト用に起動したEchoインスタンスに対してリクエストを送信し、レスポンスを取得します。
func (b *EchoTestClient) Serve() *httptest.ResponseRecorder {
	req, rec, _ := b.Build()
	b.e.ServeHTTP(rec, req)
	return rec
}
