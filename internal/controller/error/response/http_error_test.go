package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

func TestLookupErrorMetaByHTTPStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   int
		want httpErrorMeta
	}{
		{
			name: "既知のステータスコードの場合、対応するエラーを返す",
			in:   http.StatusForbidden,
			want: httpErrorMeta{Status: http.StatusForbidden, Code: codeAccessDenied, Message: errorMessageAccessDenied},
		},
		{
			name: "未知のステータスコードの場合、内部サーバーエラーとして扱う",
			in:   999,
			want: httpErrorMeta{Status: http.StatusInternalServerError, Code: codeInternalError, Message: errorMessageInternalError},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, lookupErrorMetaByHTTPStatus(tt.in))
		})
	}
}

func TestLookupErrorMetaByAppError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   error
		want httpErrorMeta
	}{
		{
			name: "ErrInvalidArgumentの場合、BadRequestが返される",
			in:   apperror.ErrInvalidArgument,
			want: httpErrorMeta{Status: http.StatusBadRequest, Code: codeBadRequest, Message: errorMessageBadRequest},
		},
		{
			name: "ErrValidationの場合、UnprocessableEntityが返される",
			in:   apperror.ErrValidation,
			want: httpErrorMeta{Status: http.StatusUnprocessableEntity, Code: codeValidationFailed, Message: errorMessageValidationFailed},
		},
		{
			name: "ErrUnauthenticatedの場合、Unauthorizedが返される",
			in:   apperror.ErrUnauthenticated,
			want: httpErrorMeta{Status: http.StatusUnauthorized, Code: codeUnauthorized, Message: errorMessageUnauthorized},
		},
		{
			name: "ErrPermissionDeniedの場合、Forbiddenが返される",
			in:   apperror.ErrPermissionDenied,
			want: httpErrorMeta{Status: http.StatusForbidden, Code: codeAccessDenied, Message: errorMessageAccessDenied},
		},
		{
			name: "ErrNotFoundの場合、NotFoundが返される",
			in:   apperror.ErrNotFound,
			want: httpErrorMeta{Status: http.StatusNotFound, Code: codeNotFound, Message: errorMessageNotFound},
		},
		{
			name: "ラップされたErrNotFoundでもアンラップしてNotFoundが返される",
			in:   xerrors.Wrap(apperror.ErrNotFound, "repository: find user"),
			want: httpErrorMeta{Status: http.StatusNotFound, Code: codeNotFound, Message: errorMessageNotFound},
		},
		{
			name: "ErrConflictの場合、Conflictが返される",
			in:   apperror.ErrConflict,
			want: httpErrorMeta{Status: http.StatusConflict, Code: codeResourceConflict, Message: errorMessageResourceConflict},
		},
		{
			name: "ErrUnavailableの場合、ServiceUnavailableが返される",
			in:   apperror.ErrUnavailable,
			want: httpErrorMeta{Status: http.StatusServiceUnavailable, Code: codeServiceUnavailable, Message: errorMessageServiceUnavailable},
		},
		{
			name: "ErrUnimplementedの場合、NotImplementedが返される",
			in:   apperror.ErrUnimplemented,
			want: httpErrorMeta{Status: http.StatusNotImplemented, Code: codeNotImplemented, Message: errorMessageNotImplemented},
		},
		{
			name: "ErrTooManyRequestsの場合、TooManyRequestsが返される",
			in:   apperror.ErrTooManyRequests,
			want: httpErrorMeta{Status: http.StatusTooManyRequests, Code: codeTooManyRequests, Message: errorMessageTooManyRequests},
		},
		{
			name: "ErrInternalの場合、InternalServerErrorが返される",
			in:   apperror.ErrInternal,
			want: httpErrorMeta{Status: http.StatusInternalServerError, Code: codeInternalError, Message: errorMessageInternalError},
		},
		{
			name: "未定義のエラーの場合、InternalServerErrorが返される",
			in:   errors.New("unknown error"),
			want: httpErrorMeta{Status: http.StatusInternalServerError, Code: codeInternalError, Message: errorMessageInternalError},
		},
		{
			name: "nilエラーの場合、InternalServerErrorが返される",
			in:   nil,
			want: httpErrorMeta{Status: http.StatusInternalServerError, Code: codeInternalError, Message: errorMessageInternalError},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, lookupErrorMetaByAppError(tt.in))
		})
	}
}

func TestErrorMetaLiteralContract(t *testing.T) {
	t.Parallel()

	t.Run("403のCodeとMessageのリテラル値を契約として固定する", func(t *testing.T) {
		t.Parallel()
		meta := lookupErrorMetaByAppError(apperror.ErrPermissionDenied)
		assert.Equal(t, "ACCESS_DENIED", meta.Code)
		assert.Equal(t, "この操作を行う権限がありません。", meta.Message)
	})

	t.Run("503のCodeとMessageのリテラル値を契約として固定する", func(t *testing.T) {
		t.Parallel()
		meta := lookupErrorMetaByAppError(apperror.ErrUnavailable)
		assert.Equal(t, "SERVICE_UNAVAILABLE", meta.Code)
		assert.Equal(t, "現在この機能はご利用いただけません。しばらくしてから再度お試しください。", meta.Message)
	})
}
