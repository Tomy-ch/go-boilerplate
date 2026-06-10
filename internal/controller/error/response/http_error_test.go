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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知のステータスコードの場合、対応するエラーを返す", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusForbidden, Code: codeAccessDenied, Message: errorMessageAccessDenied}
			assert.Equal(t, want, lookupErrorMetaByHTTPStatus(http.StatusForbidden))
		})

		t.Run("未知のステータスコードの場合、内部サーバーエラーとして扱う", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusInternalServerError, Code: codeInternalError, Message: errorMessageInternalError}
			assert.Equal(t, want, lookupErrorMetaByHTTPStatus(999))
		})
	})
}

func TestLookupErrorMetaByAppError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ErrInvalidArgumentの場合、BadRequestが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusBadRequest, Code: codeBadRequest, Message: errorMessageBadRequest}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrInvalidArgument))
		})

		t.Run("ErrValidationの場合、UnprocessableEntityが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusUnprocessableEntity, Code: codeValidationFailed, Message: errorMessageValidationFailed}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrValidation))
		})

		t.Run("ErrUnauthenticatedの場合、Unauthorizedが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusUnauthorized, Code: codeUnauthorized, Message: errorMessageUnauthorized}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrUnauthenticated))
		})

		t.Run("ErrPermissionDeniedの場合、Forbiddenが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusForbidden, Code: codeAccessDenied, Message: errorMessageAccessDenied}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrPermissionDenied))
		})

		t.Run("ErrNotFoundの場合、NotFoundが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusNotFound, Code: codeNotFound, Message: errorMessageNotFound}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrNotFound))
		})

		t.Run("ラップされたErrNotFoundでもアンラップしてNotFoundが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusNotFound, Code: codeNotFound, Message: errorMessageNotFound}
			assert.Equal(t, want, lookupErrorMetaByAppError(xerrors.Wrap(apperror.ErrNotFound, "repository: find user")))
		})

		t.Run("ErrConflictの場合、Conflictが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusConflict, Code: codeResourceConflict, Message: errorMessageResourceConflict}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrConflict))
		})

		t.Run("ErrUnavailableの場合、ServiceUnavailableが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusServiceUnavailable, Code: codeServiceUnavailable, Message: errorMessageServiceUnavailable}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrUnavailable))
		})

		t.Run("ErrUnimplementedの場合、NotImplementedが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusNotImplemented, Code: codeNotImplemented, Message: errorMessageNotImplemented}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrUnimplemented))
		})

		t.Run("ErrTooManyRequestsの場合、TooManyRequestsが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusTooManyRequests, Code: codeTooManyRequests, Message: errorMessageTooManyRequests}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrTooManyRequests))
		})

		t.Run("ErrInternalの場合、InternalServerErrorが返される", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusInternalServerError, Code: codeInternalError, Message: errorMessageInternalError}
			assert.Equal(t, want, lookupErrorMetaByAppError(apperror.ErrInternal))
		})

		t.Run("未定義のエラーの場合、InternalServerErrorにフォールバックする", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusInternalServerError, Code: codeInternalError, Message: errorMessageInternalError}
			assert.Equal(t, want, lookupErrorMetaByAppError(errors.New("unknown error")))
		})

		t.Run("nilエラーの場合、InternalServerErrorにフォールバックする", func(t *testing.T) {
			t.Parallel()
			want := httpErrorMeta{Status: http.StatusInternalServerError, Code: codeInternalError, Message: errorMessageInternalError}
			assert.Equal(t, want, lookupErrorMetaByAppError(nil))
		})
	})
}

func TestErrorMetaLiteralContract(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
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
	})
}
