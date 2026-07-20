package httpclient

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_statusToAppError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("100はエラーなし_1xxはdefault分岐", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, statusToAppError(http.StatusContinue))
		})

		t.Run("200はエラーなし", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, statusToAppError(http.StatusOK))
		})

		t.Run("204はエラーなし", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, statusToAppError(http.StatusNoContent))
		})

		t.Run("301はErrUnavailable_非追従リダイレクト", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusMovedPermanently), apperror.ErrUnavailable)
		})

		t.Run("302はErrUnavailable_非追従リダイレクト", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusFound), apperror.ErrUnavailable)
		})

		t.Run("400はErrInvalidArgument", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusBadRequest), apperror.ErrInvalidArgument)
		})

		t.Run("401はErrUnauthenticated", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusUnauthorized), apperror.ErrUnauthenticated)
		})

		t.Run("403はErrPermissionDenied", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusForbidden), apperror.ErrPermissionDenied)
		})

		t.Run("404はErrNotFound", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusNotFound), apperror.ErrNotFound)
		})

		t.Run("409はErrConflict", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusConflict), apperror.ErrConflict)
		})

		t.Run("422はErrValidation", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusUnprocessableEntity), apperror.ErrValidation)
		})

		t.Run("429はErrTooManyRequests", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusTooManyRequests), apperror.ErrTooManyRequests)
		})

		t.Run("418など他の4xxはErrInvalidArgument", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusTeapot), apperror.ErrInvalidArgument)
		})

		t.Run("500はErrUnavailable", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusInternalServerError), apperror.ErrUnavailable)
		})

		t.Run("503はErrUnavailable", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, statusToAppError(http.StatusServiceUnavailable), apperror.ErrUnavailable)
		})
	})
}

func Test_normalizeTransportError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctxキャンセルはErrCanceledに写像する", func(t *testing.T) {
			t.Parallel()
			got := normalizeTransportError(context.Canceled)
			require.ErrorIs(t, got, apperror.ErrCanceled)
		})

		t.Run("url_ErrorでラップされたctxキャンセルもErrCanceledに写像しURLをredactする", func(t *testing.T) {
			t.Parallel()

			urlErr := &url.Error{
				Op:  "Get",
				URL: "https://api.example.com/rates?token=secret123",
				Err: context.Canceled,
			}
			got := normalizeTransportError(urlErr)

			require.ErrorIs(t, got, apperror.ErrCanceled)
			assert.Contains(t, got.Error(), "api.example.com")
			assert.NotContains(t, got.Error(), "secret123")
		})

		t.Run("deadline超過はErrUnavailableに写像する", func(t *testing.T) {
			t.Parallel()
			got := normalizeTransportError(context.DeadlineExceeded)
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})

		t.Run("その他のtransport失敗はErrUnavailableに写像する", func(t *testing.T) {
			t.Parallel()
			got := normalizeTransportError(xerrors.New("dial tcp: connection refused"))
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})
	})
}

func Test_redactErrMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("url_Errorはホストのみへredactしクエリを落とす", func(t *testing.T) {
			t.Parallel()

			urlErr := &url.Error{
				Op:  "Get",
				URL: "https://api.example.com/rates?token=secret123&base=USD",
				Err: xerrors.New("dial tcp 93.184.216.34:443: connect: connection refused"),
			}

			msg := redactErrMessage(urlErr)
			assert.Contains(t, msg, "api.example.com")
			assert.NotContains(t, msg, "secret123")
			assert.NotContains(t, msg, "token")
		})

		t.Run("url_Errorのuserinfo_userpass_はredactしホストパスのみ残す", func(t *testing.T) {
			t.Parallel()

			urlErr := &url.Error{ //nolint:gosec // G101: redact 対象を検証するための擬似 userinfo 付き URL（実認証情報ではない）
				Op:  "Get",
				URL: "https://user:pass@api.example.com/rates",
				Err: xerrors.New("connection refused"),
			}

			msg := redactErrMessage(urlErr)
			assert.Contains(t, msg, "api.example.com/rates")
			assert.NotContains(t, msg, "user")
			assert.NotContains(t, msg, "pass")
		})

		t.Run("url_Errorのfragment_section_はredactする", func(t *testing.T) {
			t.Parallel()

			urlErr := &url.Error{
				Op:  "Get",
				URL: "https://api.example.com/rates#section-secret",
				Err: xerrors.New("connection refused"),
			}

			msg := redactErrMessage(urlErr)
			assert.Contains(t, msg, "api.example.com/rates")
			assert.NotContains(t, msg, "section-secret")
		})

		t.Run("url_Error以外はそのまま返す", func(t *testing.T) {
			t.Parallel()

			msg := redactErrMessage(xerrors.New("plain error"))
			assert.Equal(t, "plain error", msg)
		})
	})
}

func Test_statusClass(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("100番台は1xx", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "1xx", statusClass(http.StatusContinue))
		})

		t.Run("200番台は2xx", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "2xx", statusClass(http.StatusOK))
		})

		t.Run("300番台は3xx", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "3xx", statusClass(http.StatusFound))
		})

		t.Run("400番台は4xx", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "4xx", statusClass(http.StatusNotFound))
		})

		t.Run("500番台は5xx", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "5xx", statusClass(http.StatusBadGateway))
		})
	})
}

func Test_redactURL(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クエリ_userinfo_fragmentを除去しscheme_host_pathを残す", func(t *testing.T) {
			t.Parallel()

			out := redactURL("https://user:pass@api.example.com/rates?token=secret#frag")

			assert.Equal(t, "https://api.example.com/rates", out)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パース不能なURLはupstreamを返す", func(t *testing.T) {
			t.Parallel()

			out := redactURL("://not a url\x7f")

			assert.Equal(t, "upstream", out)
		})
	})
}
