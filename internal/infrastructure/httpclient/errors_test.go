package httpclient

import (
	"context"
	"errors"
	"net/http"
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
			"200はエラーなし":                     {status: http.StatusOK, wantNil: true},
			"204はエラーなし":                     {status: http.StatusNoContent, wantNil: true},
			"301はエラーなし":                     {status: http.StatusMovedPermanently, wantNil: true},
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
