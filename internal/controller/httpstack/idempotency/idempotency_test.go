package idempotency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/usecase/boundary/auth"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	idempotencyuc "go-boilerplate/internal/usecase/idempotency"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	fingerprintLen = 32
	// testPath は、テスト対象の POST パスです。
	testPath = "/v1/users"
	// sentinel は、後段ハンドラが呼ばれたことを示す戻り値です。
	sentinel = "SENTINEL"
)

type spyRequest struct {
	Name string `json:"name"`
}

// newEcho は、テスト用の echo.Context（POST /v1/users）を生成します。key 非空ならヘッダを付与し、
// withAuthn なら subject を持つ Authn を ctx に仕込みます。
func newEcho(key string, withAuthn bool, subject string) echo.Context {
	ctx := context.Background()
	if withAuthn {
		ctx = ctxhelper.WithAuthn(ctx)
		a, _ := auth.New(subject, "test", nil, nil)
		ctxhelper.SetAuthn(ctx, *a)
	}
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, testPath, nil)
	if key != "" {
		req.Header.Set(headerName, key)
	}
	return echo.New().NewContext(req, httptest.NewRecorder())
}

func TestMiddleware_handle(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘッダ無しは素通しし後段をそのまま呼ぶ", func(t *testing.T) {
			t.Parallel()
			called := false
			next := NextFunc(func(echo.Context, any) (any, error) {
				called = true
				return sentinel, nil
			})
			ec := newEcho("", true, "user-1")

			res, err := Middleware()(next, "PostUsers")(ec, spyRequest{})

			require.NoError(t, err)
			assert.True(t, called)
			assert.Equal(t, sentinel, res)
		})

		t.Run("認証が無ければ冪等性は発動せず素通しする", func(t *testing.T) {
			t.Parallel()
			called := false
			next := NextFunc(func(echo.Context, any) (any, error) {
				called = true
				return sentinel, nil
			})
			ec := newEcho("key-1", false, "")

			res, err := Middleware()(next, "PostUsers")(ec, spyRequest{})

			require.NoError(t, err)
			assert.True(t, called)
			assert.Equal(t, sentinel, res)
		})

		t.Run("有効キー+認証ありなら Request を ctx に載せて後段へ渡す", func(t *testing.T) {
			t.Parallel()
			called := false
			next := NextFunc(func(echo.Context, any) (any, error) {
				called = true
				return sentinel, nil
			})
			ec := newEcho("key-abc", true, "user-9")

			_, err := Middleware()(next, "PostUsers")(ec, spyRequest{Name: "alice"})
			require.NoError(t, err)
			require.True(t, called)

			// middleware は ec.SetRequest で stash 済み。その ctx を Run に渡し Claim の引数を検証する。
			gotCtx := ec.Request().Context()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			txm := mock_tx.NewMockManager(ctrl)
			clk := mock_clock.NewMockClock(ctrl)
			clk.EXPECT().Now().Return(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)).AnyTimes()
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })

			var got idempotencybndry.ClaimParams
			store.EXPECT().Claim(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, p idempotencybndry.ClaimParams) (bool, error) {
					got = p
					return true, nil
				})
			store.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(nil)

			deps := idempotencyuc.Deps{Txm: txm, Store: store, Clock: clk}
			_, _, runErr := idempotencyuc.Run(gotCtx, deps, 201,
				func(context.Context) (string, error) { return "ok", nil })
			require.NoError(t, runErr)

			assert.Equal(t, "user-9", got.Scope)
			assert.Equal(t, "key-abc", got.Key)
			assert.Equal(t, http.MethodPost, got.Method)
			assert.Equal(t, testPath, got.Path)
			assert.Len(t, got.Fingerprint, fingerprintLen)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("255文字超のキーは400で後段を呼ばない", func(t *testing.T) {
			t.Parallel()
			called := false
			next := NextFunc(func(echo.Context, any) (any, error) {
				called = true
				return sentinel, nil
			})
			ec := newEcho(strings.Repeat("a", 256), true, "user-1")

			_, err := Middleware()(next, "PostUsers")(ec, spyRequest{})

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.False(t, called)
		})

		t.Run("非印字ASCIIを含むキーは400で後段を呼ばない", func(t *testing.T) {
			t.Parallel()
			called := false
			next := NextFunc(func(echo.Context, any) (any, error) {
				called = true
				return sentinel, nil
			})
			ec := newEcho("key with space", true, "user-1")

			_, err := Middleware()(next, "PostUsers")(ec, spyRequest{})

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.False(t, called)
		})
	})
}

func Test_validateKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系_印字可能ASCIIは通る", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateKey("Idem-Key_123.~"))
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		cases := map[string]string{
			"長すぎ":        strings.Repeat("x", 256),
			"空白(0x20)":   "a b",
			"制御文字(0x1f)": "a\x1fb",
			"DEL(0x7f)":  "a\x7fb",
			"マルチバイト":     "あ",
		}
		for name, key := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, validateKey(key), apperror.ErrInvalidArgument)
			})
		}
	})
}

func Test_fingerprint(t *testing.T) {
	t.Parallel()

	base := fingerprint(http.MethodPost, testPath, spyRequest{Name: "alice"})

	t.Run("SHA-256の32バイトを返す", func(t *testing.T) {
		t.Parallel()
		assert.Len(t, base, fingerprintLen)
	})

	t.Run("同一入力は同一指紋(決定的)", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, base, fingerprint(http.MethodPost, testPath, spyRequest{Name: "alice"}))
	})

	t.Run("リクエストボディが異なれば指紋も異なる", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, base, fingerprint(http.MethodPost, testPath, spyRequest{Name: "bob"}))
	})

	t.Run("methodが異なれば指紋も異なる", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, base, fingerprint(http.MethodPut, testPath, spyRequest{Name: "alice"}))
	})

	t.Run("pathが異なれば指紋も異なる", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, base, fingerprint(http.MethodPost, "/v1/orders", spyRequest{Name: "alice"}))
	})
}
