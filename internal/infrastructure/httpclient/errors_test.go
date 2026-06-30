package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"go-boilerplate/internal/apperror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusToAppError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		cases := map[string]struct {
			status  int
			want    error
			wantNil bool
		}{
			"100はエラーなし_1xxはdefault分岐":       {status: http.StatusContinue, wantNil: true},
			"200はエラーなし":                     {status: http.StatusOK, wantNil: true},
			"204はエラーなし":                     {status: http.StatusNoContent, wantNil: true},
			"301はErrUnavailable_非追従リダイレクト":  {status: http.StatusMovedPermanently, want: apperror.ErrUnavailable},
			"302はErrUnavailable_非追従リダイレクト":  {status: http.StatusFound, want: apperror.ErrUnavailable},
			"400はErrInvalidArgument":        {status: http.StatusBadRequest, want: apperror.ErrInvalidArgument},
			"401はErrUnauthenticated":        {status: http.StatusUnauthorized, want: apperror.ErrUnauthenticated},
			"403はErrPermissionDenied":       {status: http.StatusForbidden, want: apperror.ErrPermissionDenied},
			"404はErrNotFound":               {status: http.StatusNotFound, want: apperror.ErrNotFound},
			"409はErrConflict":               {status: http.StatusConflict, want: apperror.ErrConflict},
			"422はErrValidation":             {status: http.StatusUnprocessableEntity, want: apperror.ErrValidation},
			"429はErrTooManyRequests":        {status: http.StatusTooManyRequests, want: apperror.ErrTooManyRequests},
			"418など他の4xxはErrInvalidArgument": {status: http.StatusTeapot, want: apperror.ErrInvalidArgument},
			"500はErrUnavailable":            {status: http.StatusInternalServerError, want: apperror.ErrUnavailable},
			"503はErrUnavailable":            {status: http.StatusServiceUnavailable, want: apperror.ErrUnavailable},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				got := statusToAppError(tc.status)
				if tc.wantNil {
					require.NoError(t, got)
					return
				}
				require.ErrorIs(t, got, tc.want)
			})
		}
	})
}

func TestNormalizeTransportError(t *testing.T) {
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
			got := normalizeTransportError(errors.New("dial tcp: connection refused"))
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})
	})
}

func TestRedactErrMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("url_Errorはホストのみへredactしクエリを落とす", func(t *testing.T) {
			t.Parallel()

			urlErr := &url.Error{
				Op:  "Get",
				URL: "https://api.example.com/rates?token=secret123&base=USD",
				Err: errors.New("dial tcp 93.184.216.34:443: connect: connection refused"),
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
				Err: errors.New("connection refused"),
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
				Err: errors.New("connection refused"),
			}

			msg := redactErrMessage(urlErr)
			assert.Contains(t, msg, "api.example.com/rates")
			assert.NotContains(t, msg, "section-secret")
		})

		t.Run("url_Error以外はそのまま返す", func(t *testing.T) {
			t.Parallel()

			msg := redactErrMessage(errors.New("plain error"))
			assert.Equal(t, "plain error", msg)
		})
	})
}

func TestStatusClass(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		cases := map[string]struct {
			status int
			want   string
		}{
			"100番台は1xx": {status: http.StatusContinue, want: "1xx"},
			"200番台は2xx": {status: http.StatusOK, want: "2xx"},
			"300番台は3xx": {status: http.StatusFound, want: "3xx"},
			"400番台は4xx": {status: http.StatusNotFound, want: "4xx"},
			"500番台は5xx": {status: http.StatusBadGateway, want: "5xx"},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tc.want, statusClass(tc.status))
			})
		}
	})
}
