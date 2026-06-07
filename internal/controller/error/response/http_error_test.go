package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/apperror"
)

func TestLookupErrorMetaByHTTPStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("既知のステータスコードの場合、対応するエラーを返す", func(t *testing.T) {
			t.Parallel()
			httpStatus := http.StatusForbidden

			expected := httpErrorMeta{
				Status:  httpStatus,
				Code:    codeAccessDenied,
				Message: errorMessageAccessDenied,
			}

			actual := lookupErrorMetaByHTTPStatus(httpStatus)

			assert.Equal(t, expected, actual)
		})

		t.Run("未知のステータスコードの場合、内部サーバーエラーとして扱う", func(t *testing.T) {
			t.Parallel()
			httpStatus := 999

			expected := httpErrorMeta{
				Status:  http.StatusInternalServerError,
				Code:    codeInternalError,
				Message: errorMessageInternalError,
			}

			actual := lookupErrorMetaByHTTPStatus(httpStatus)

			assert.Equal(t, expected, actual)
		})
	})
}

func TestLookupErrorMetaByAppError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ErrInvalidArgumentの場合、BadRequestが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusBadRequest,
				Code:    codeBadRequest,
				Message: errorMessageBadRequest,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrInvalidArgument)
			assert.Equal(t, expected, actual)
		})

		t.Run("ErrValidationの場合、UnprocessableEntityが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusUnprocessableEntity,
				Code:    codeValidationFailed,
				Message: errorMessageValidationFailed,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrValidation)
			assert.Equal(t, expected, actual)
		})

		t.Run("ErrUnauthenticatedの場合、unauthorizedが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusUnauthorized,
				Code:    codeUnauthorized,
				Message: errorMessageUnauthorized,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrUnauthenticated)
			assert.Equal(t, expected, actual)
		})

		t.Run("ErrPermissionDeniedの場合、Forbiddenが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusForbidden,
				Code:    codeAccessDenied,
				Message: errorMessageAccessDenied,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrPermissionDenied)
			assert.Equal(t, expected, actual)
		})

		t.Run("ErrNotFoundの場合、NotFoundが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusNotFound,
				Code:    codeNotFound,
				Message: errorMessageNotFound,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrNotFound)
			assert.Equal(t, expected, actual)
		})

		t.Run("ErrConflictの場合、Conflictが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusConflict,
				Code:    codeResourceConflict,
				Message: errorMessageResourceConflict,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrConflict)
			assert.Equal(t, expected, actual)
		})

		t.Run("ErrUnavailableの場合、ServiceUnavailableが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusServiceUnavailable,
				Code:    codeNotAvailable,
				Message: errorMessageNotAvailable,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrUnavailable)
			assert.Equal(t, expected, actual)
		})

		t.Run("ErrUnimplementedの場合、NotImplementedが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusNotImplemented,
				Code:    codeNotAvailable,
				Message: errorMessageNotAvailable,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrUnimplemented)
			assert.Equal(t, expected, actual)
		})

		t.Run("ErrTooManyRequestsの場合、TooManyRequestsが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusTooManyRequests,
				Code:    codeTooManyRequests,
				Message: errorMessageTooManyRequests,
			}

			actual := lookupErrorMetaByAppError(apperror.ErrTooManyRequests)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未定義のエラーの場合、InternalServerErrorが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusInternalServerError,
				Code:    codeInternalError,
				Message: errorMessageInternalError,
			}

			actual := lookupErrorMetaByAppError(errors.New("unknown error"))
			assert.Equal(t, expected, actual)
		})

		t.Run("nilエラーの場合、InternalServerErrorが返される", func(t *testing.T) {
			expected := httpErrorMeta{
				Status:  http.StatusInternalServerError,
				Code:    codeInternalError,
				Message: errorMessageInternalError,
			}

			actual := lookupErrorMetaByAppError(nil)
			assert.Equal(t, expected, actual)
		})
	})
}
