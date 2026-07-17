// Package integration は統合テスト用のパッケージです。
package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/ctxhelper"
	responsegen "go-boilerplate/internal/controller/error/response/gen"
	"go-boilerplate/internal/controller/httpstack/errorhandler"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Server struct {
	t       *testing.T
	e       *echo.Echo
	ts      *httptest.Server
	baseURL string
	client  *http.Client
}

// MakeAvailableUserID は、指定したユーザーIDで認証された状態を模擬するミドルウェアをEchoに追加し、
// その認証情報を含むHTTPヘッダーを返します。
func MakeAvailableUserID(t *testing.T, e *echo.Echo, id uuid.UUID) http.Header {
	t.Helper()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if a, err := auth.New(id.String(), auth.ProviderMock, nil, nil); err == nil {
				req := c.Request()
				ctx := ctxhelper.WithAuthn(req.Context())
				ctxhelper.SetAuthn(ctx, *a)
				c.SetRequest(req.WithContext(ctx))
			}
			return next(c)
		}
	})

	h := http.Header{}
	h.Set("Authorization", "Bearer debug:"+id.String())
	return h
}

// StartServer は Echo を起動し、httptest.Server を返す。
//
// e はハンドラが設定済みの状態で渡すこと。
func StartServer(t *testing.T, e *echo.Echo) *Server {
	t.Helper()

	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)

	return &Server{
		t:       t,
		e:       e,
		ts:      ts,
		baseURL: ts.URL,
		client:  &http.Client{Timeout: 3 * time.Second},
	}
}

// StopServer は、サーバーを停止します。
func (s *Server) StopServer() { s.t.Helper(); s.t.Cleanup(s.ts.Close) }

// Do は、任意メソッド/パス/ボディでHTTPを実行する。
func (s *Server) Do(
	method, path string, reqBody io.Reader, contentType string, headers http.Header,
) *http.Response {
	s.t.Helper()

	req, err := http.NewRequestWithContext(s.t.Context(), method, s.baseURL+path, reqBody)
	require.NoError(s.t, err)

	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := s.client.Do(req)
	require.NoError(s.t, err)
	defer s.t.Cleanup(func() {
		require.NoError(s.t, res.Body.Close())
	})

	return res
}

// DoJSON は、JSONで送受信するユーティリティ。
func (s *Server) DoJSON(
	method, path string, reqBody any, headers http.Header,
) *http.Response {
	s.t.Helper()

	var r io.Reader
	if reqBody != nil {
		buf, err := json.Marshal(reqBody)
		require.NoError(s.t, err)
		r = bytes.NewReader(buf)
	}

	return s.Do(method, path, r, "application/json", headers)
}

// AssertJSONResponseType は、200 / JSON Content-Type を確認したうえで、
// レスポンスボディが型 T にデコード可能であることを検証する到達確認ユーティリティ。
//
// integration テストは HTTP 境界（router → middleware → handler → シリアライズ）の到達と
// レスポンスの型整合のみを検証する。フィールド値の正しさは controller のユニットテストが
// 独立オラクルで担保するため、ここでは値比較を行わない。
func AssertJSONResponseType[T any](t *testing.T, actualResponse *http.Response) {
	t.Helper()

	resBody, err := io.ReadAll(actualResponse.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, actualResponse.StatusCode)
	assert.Contains(t, actualResponse.Header.Get(echo.HeaderContentType), "application/json")

	var actualObj T
	require.NoError(t, json.Unmarshal(resBody, &actualObj), "返却された型が期待された型と一致しません。型引数に期待される型を指定してください。")
}

// UseAppErrorHandler は、本番相当の HTTPErrorHandler を Echo に登録する。
//
// 既定の echo.New() は標準のエラーハンドラを持つため、異常系で apperror → HTTP ステータスの
// マッピングを実経路で検証するには、production と同じハンドラを配線する必要がある。
func UseAppErrorHandler(t *testing.T, e *echo.Echo) {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)

	spec, err := validator.GetValidator()
	require.NoError(t, err)
	policy, err := errorhandler.NewOpenAPIDetailPolicy(spec)
	require.NoError(t, err)

	errorhandler.New(e, policy, logging.NewTestLogger(t), lf, obsCfg)
}

// AssertErrorResponse は、異常系レスポンスの HTTP ステータスが wantStatus と一致し、
// ボディが JSON のエラーレスポンス（ErrorResponse）としてシリアライズされていることを検証する。
//
// integration テストは HTTP 境界の関心事である「apperror → ステータスコードのマッピング」と
// エラーボディの形のみを検証する。Code/Message の値の正しさは controller のユニットテストが担う。
func AssertErrorResponse(t *testing.T, actualResponse *http.Response, wantStatus int) {
	t.Helper()

	AssertErrorResponseBody(t, actualResponse, wantStatus)
}

// AssertErrorResponseBody は、[AssertErrorResponse] と同じ検証を行ったうえで、
// デコード済みの ErrorResponse を返します。details 等のボディ内容まで検証する場合に使います。
func AssertErrorResponseBody(t *testing.T, actualResponse *http.Response, wantStatus int) responsegen.ErrorResponseWithDetails {
	t.Helper()

	resBody, err := io.ReadAll(actualResponse.Body)
	require.NoError(t, err)

	assert.Equal(t, wantStatus, actualResponse.StatusCode)
	assert.Contains(t, actualResponse.Header.Get(echo.HeaderContentType), "application/json")

	var errResp responsegen.ErrorResponseWithDetails
	require.NoError(t, json.Unmarshal(resBody, &errResp), "エラーレスポンスが ErrorResponse 形式でシリアライズされていません。")
	assert.NotEmpty(t, errResp.Code)
	return errResp
}
