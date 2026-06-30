package timeout

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	exec := func(t *testing.T, timeout time.Duration, handler echo.HandlerFunc) (int, error) {
		t.Helper()
		e := echo.New()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		err := Middleware(timeout)(handler)(c)
		return rec.Code, err
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限内に完了するハンドラはそのまま通過する", func(t *testing.T) {
			t.Parallel()

			var deadlineSet bool
			code, err := exec(t, time.Second, func(c echo.Context) error {
				_, deadlineSet = c.Request().Context().Deadline()
				return c.NoContent(http.StatusOK)
			})

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, code)
			assert.True(t, deadlineSet, "リクエスト context に deadline が設定されること")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限超過時はErrUnavailableにwrapして返す", func(t *testing.T) {
			t.Parallel()

			// ハンドラは deadline を尊重して ctx 完了後に返る。ContextTimeout が超過を検知し
			// ErrorHandler 経由で apperror.ErrUnavailable を返す。
			_, err := exec(t, 20*time.Millisecond, func(c echo.Context) error {
				<-c.Request().Context().Done()
				return c.Request().Context().Err()
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("期限超過前に非タイムアウトエラーを返した場合は元のエラーを保持する", func(t *testing.T) {
			t.Parallel()

			// ハンドラはタイムアウト発火前に apperror.ErrNotFound を返す。ErrorHandler は
			// DeadlineExceeded 以外を素通しするため、ErrUnavailable へは変換されない。
			_, err := exec(t, time.Second, func(echo.Context) error {
				return apperror.ErrNotFound
			})

			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}
