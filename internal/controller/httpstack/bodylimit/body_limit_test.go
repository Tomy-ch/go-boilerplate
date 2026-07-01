package bodylimit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	// exec は、limitMB 上限のミドルウェア配下で body を POST し、(handler 実行有無, 返り error) を返す。
	exec := func(t *testing.T, limitMB int, body []byte) (bool, error) {
		t.Helper()
		e := echo.New()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handled := false
		handler := func(c echo.Context) error {
			_, _ = io.ReadAll(c.Request().Body) // BodyLimit は reader をラップするので読み切る
			handled = true
			return c.NoContent(http.StatusOK)
		}
		return handled, Middleware(limitMB)(handler)(c)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限内のボディは通過しハンドラが実行される", func(t *testing.T) {
			t.Parallel()

			handled, err := exec(t, 1, bytes.Repeat([]byte("a"), 1000)) // 1KB < 1MB
			require.NoError(t, err)
			assert.True(t, handled)
		})

		t.Run("ちょうど上限値のボディは通過しハンドラが実行される", func(t *testing.T) {
			t.Parallel()

			// limit 1MB=1,000,000 byte ちょうどは上限以下として通過する（境界の通過側）。
			handled, err := exec(t, 1, bytes.Repeat([]byte("a"), 1_000_000))
			require.NoError(t, err)
			assert.True(t, handled)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限超過のボディは413で弾かれハンドラは実行されない", func(t *testing.T) {
			t.Parallel()

			// limit 1MB=1,000,000 byte に対し 2,000,000 byte を送る。
			handled, err := exec(t, 1, bytes.Repeat([]byte("a"), 2_000_000))

			require.Error(t, err)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusRequestEntityTooLarge, he.Code)
			assert.False(t, handled)
		})

		t.Run("上限を1バイト超えるボディは413で弾かれハンドラは実行されない", func(t *testing.T) {
			t.Parallel()

			// limit 1MB=1,000,000 byte に対し 1,000,001 byte（境界の超過側）を送る。
			handled, err := exec(t, 1, bytes.Repeat([]byte("a"), 1_000_001))

			require.Error(t, err)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusRequestEntityTooLarge, he.Code)
			assert.False(t, handled)
		})

		t.Run("Content-Length不明でも実読み取りが上限超過なら413になる", func(t *testing.T) {
			t.Parallel()

			// io.NopCloser で包むと Content-Length が確定せず、BodyLimit は ContentLength の
			// 早期判定を通過し、実読み取り中の上限チェックで 413 を返す。
			e := echo.New()
			body := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), 2_000_000)))
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handled := false
			handler := func(c echo.Context) error {
				_, err := io.ReadAll(c.Request().Body) // 実読み取りで上限超過し 413 を返す
				handled = true
				return err
			}
			err := Middleware(1)(handler)(c)

			require.Error(t, err)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusRequestEntityTooLarge, he.Code)
			assert.True(t, handled, "実読み取り経路を通すためハンドラ自体は実行される")
		})

		t.Run("limitMBが0以下の場合はパニックする", func(t *testing.T) {
			t.Parallel()

			assert.Panics(t, func() { Middleware(0) })
			assert.Panics(t, func() { Middleware(-1) })
		})
	})
}
