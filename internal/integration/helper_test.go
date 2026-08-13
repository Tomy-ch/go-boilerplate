// Package integration は統合テスト用のパッケージです。
package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect" //nolint:depguard // テスト専用。宣言されたスライスフィールドを JSON と突き合わせ「配列は null でなく [] を返す」シリアライズ契約を型横断で検証するのに reflect が必須（本番コードは reflect を使わない）。
	"strconv"
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/ctxhelper"
	responsegen "go-boilerplate/internal/controller/error/response/gen"
	"go-boilerplate/internal/controller/httpstack/errorhandler"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/instrumentation"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
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

	// Authn の生成はテスト goroutine で済ませる。ミドルウェアはサーバー goroutine で動くため、
	// そちらで require を呼ぶと FailNow がテスト本体に届かない。
	a, err := auth.New(id.String(), auth.IssuerMock, nil, nil)
	require.NoError(t, err)
	authn, err := a.WithUserID(id)
	require.NoError(t, err)

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			ctx := ctxhelper.WithAuthn(req.Context())
			ctxhelper.SetAuthn(ctx, *authn)
			c.SetRequest(req.WithContext(ctx))
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
// 値比較は行わない（境界と値の切り分けは README を参照）。
//
// 加えてシリアライズの形の検査として、T にスライスとして宣言されたフィールドが
// JSON で null になっていないこと（空でも [] を返す API 契約）を検証する。
func AssertJSONResponseType[T any](t *testing.T, actualResponse *http.Response) {
	t.Helper()

	resBody, err := io.ReadAll(actualResponse.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, actualResponse.StatusCode)
	assert.Contains(t, actualResponse.Header.Get(echo.HeaderContentType), "application/json")

	var actualObj T
	require.NoError(t, json.Unmarshal(resBody, &actualObj), "返却された型が期待された型と一致しません。型引数に期待される型を指定してください。")

	assertDeclaredArraysNotNull(t, resBody, reflect.TypeFor[T](), "")
}

// assertDeclaredArraysNotNull は、typ 内でスライスとして宣言されたフィールドが raw の JSON 上で
// null にシリアライズされていないことを再帰的に検証する。
//
// 違反となるのはキーが存在して値が null の場合のみで、キー自体が無い（omitempty で absent）場合は
// 違反としない。[]byte は base64 文字列にシリアライズされるため対象外。
func assertDeclaredArraysNotNull(t *testing.T, raw json.RawMessage, typ reflect.Type, path string) {
	t.Helper()

	if isJSONNull(raw) {
		return
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		assertStructArraysNotNull(t, raw, typ, path)
	case reflect.Slice:
		assertSliceArraysNotNull(t, raw, typ, path)
	default:
		// スカラー等、配列を含み得ない型は検査対象外。
	}
}

// assertStructArraysNotNull は、struct 型 typ の各フィールドを JSON と突き合わせて検査する。
func assertStructArraysNotNull(t *testing.T, raw json.RawMessage, typ reflect.Type, path string) {
	t.Helper()

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		// time.Time など JSON オブジェクト以外へシリアライズされる struct は対象外。
		return
	}
	// VisibleFields はタグ無し埋め込みフィールドを昇格済みの形で列挙するため、
	// 埋め込みのインライン化を個別に扱う必要がない。
	for _, f := range reflect.VisibleFields(typ) {
		if f.Anonymous || !f.IsExported() {
			continue
		}
		name, ok := jsonFieldName(f)
		if !ok {
			continue
		}
		fieldRaw, present := fields[name]
		if !present {
			continue
		}
		fieldPath := name
		if path != "" {
			fieldPath = path + "." + name
		}
		if isDeclaredArray(f.Type) && isJSONNull(fieldRaw) {
			assert.Failf(t, "配列フィールドが null",
				"配列フィールド %q が JSON で null にシリアライズされています（[] を返す契約違反）", fieldPath)
			continue
		}
		assertDeclaredArraysNotNull(t, fieldRaw, f.Type, fieldPath)
	}
}

// assertSliceArraysNotNull は、スライス型 typ の各要素を再帰的に検査する。
func assertSliceArraysNotNull(t *testing.T, raw json.RawMessage, typ reflect.Type, path string) {
	t.Helper()

	if typ.Elem().Kind() == reflect.Uint8 {
		return // []byte は base64 文字列。
	}
	var elems []json.RawMessage
	if err := json.Unmarshal(raw, &elems); err != nil {
		return
	}
	for i, elemRaw := range elems {
		assertDeclaredArraysNotNull(t, elemRaw, typ.Elem(), path+"["+strconv.Itoa(i)+"]")
	}
}

// isDeclaredArray は、t がポインタを剥がしたうえで JSON 配列としてシリアライズされる
// スライス型（[]byte を除く）かを返す。
func isDeclaredArray(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
}

// jsonFieldName は、struct フィールドの JSON キー名を解決する。`json:"-"` のときは ok=false。
func jsonFieldName(f reflect.StructField) (string, bool) {
	tag, tagged := f.Tag.Lookup("json")
	if !tagged {
		return f.Name, true
	}
	if tag == "-" {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return f.Name, true
	}
	return name, true
}

// isJSONNull は、raw が JSON リテラル null かどうかを返す。
func isJSONNull(raw json.RawMessage) bool {
	return len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

// UseAppErrorHandler は、本番相当の HTTPErrorHandler と、それが返すエラー応答の契約が
// 成立するために必要なミドルウェアを Echo に登録する。
//
// 既定の echo.New() は標準のエラーハンドラを持つため、異常系で apperror → HTTP ステータスの
// マッピングを実経路で検証するには、production と同じハンドラを配線する必要がある。
//
// ハンドラだけを配線すると requestId が常に空になるため、requestid ミドルウェアと一体に配線する。
//
// extra には、この契約に相乗りさせたい本番ミドルウェアを DI provider から渡す。適用順は
// 本番と同じ Priority ソートが決めるので、呼び出し側は順序を書かない。
func UseAppErrorHandler(t *testing.T, e *echo.Echo, extra ...extension.UseMiddleware) {
	t.Helper()

	mws := append([]extension.UseMiddleware{instrumentation.RequestIDMiddleware().Middleware}, extra...)
	require.NoError(t, extension.ApplyUseMiddlewares(e, logging.NewTestLogger(t), mws))

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)

	spec, err := validator.GetValidator()
	require.NoError(t, err)
	detailPolicy, err := errorhandler.NewOpenAPIDetailPolicy(spec)
	require.NoError(t, err)
	allowPolicy, err := errorhandler.NewOpenAPIAllowPolicy(spec)
	require.NoError(t, err)

	errorhandler.New(e, errorhandler.Policies{Detail: detailPolicy, Allow: allowPolicy}, logging.NewTestLogger(t), lf, obsCfg)
}

// AssertErrorResponse は、異常系レスポンスの HTTP ステータスが wantStatus と一致し、
// ボディが JSON のエラーレスポンス（ErrorResponse）としてシリアライズされていることを検証する。
// [AssertJSONResponseType] と同じく境界のみを見て、Code/Message の値は比較しない。
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
	assert.Equal(t, actualResponse.Header.Get(echo.HeaderXRequestID), errResp.RequestId,
		"エラーボディの requestId がワイヤ上の X-Request-Id と食い違っています。")
	assert.NotEmpty(t, errResp.RequestId)
	return errResp
}
