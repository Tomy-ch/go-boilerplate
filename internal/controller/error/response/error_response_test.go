package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/error/response/gen"
	"go-boilerplate/pkg/xerrors"
)

func TestNewHTTPErrorFromAppError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("detailsがある場合、detailsが表示されるエラー構造体が返る", func(t *testing.T) {
			t.Parallel()

			err := xerrors.Wrap(apperror.ErrInvalidArgument, "bad request error")
			details := []string{"invalid input", "missing field"}

			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    codeBadRequest,
					Message: errorMessageBadRequest,
					Details: new(details),
				},
				HTTPStatus: http.StatusBadRequest,
				Internal:   err,
			}

			assert.Equal(t, expected, NewHTTPErrorFromAppError(err, details...))
		})

		t.Run("detailsがない場合、detailsは表示されないエラー構造体が返る", func(t *testing.T) {
			t.Parallel()

			err := xerrors.Wrap(apperror.ErrInternal, "internal server error")

			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    codeInternalError,
					Message: errorMessageInternalError,
				},
				HTTPStatus: http.StatusInternalServerError,
				Internal:   err,
			}

			assert.Equal(t, expected, NewHTTPErrorFromAppError(err))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のエラーの場合、500のエラー構造体にフォールバックする", func(t *testing.T) {
			t.Parallel()
			err := errors.New("unknown error")
			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeInternalError, Message: errorMessageInternalError},
				HTTPStatus:    http.StatusInternalServerError,
				Internal:      err,
			}
			assert.Equal(t, expected, NewHTTPErrorFromAppError(err))
		})
	})
}

func TestNewHTTPErrorFromStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知のステータスコードの場合、対応するエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeNotFound, Message: errorMessageNotFound},
				HTTPStatus:    http.StatusNotFound,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusNotFound, nil))
		})

		t.Run("詳細が渡された場合、Detailsにセットされる", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    codeResourceConflict,
					Message: errorMessageResourceConflict,
					Details: new([]string{"conflict-1"}),
				},
				HTTPStatus: http.StatusConflict,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusConflict, nil, "conflict-1"))
		})

		t.Run("errを渡した場合、Internalに格納される", func(t *testing.T) {
			t.Parallel()
			internalErr := errors.New("boom")
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeBadRequest, Message: errorMessageBadRequest},
				HTTPStatus:    http.StatusBadRequest,
				Internal:      internalErr,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusBadRequest, internalErr))
		})

		t.Run("未知のステータスコードの場合、内部エラーとして扱われる", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeInternalError, Message: errorMessageInternalError},
				HTTPStatus:    http.StatusInternalServerError,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(999, nil))
		})

		t.Run("401の場合、Unauthorizedのエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeUnauthorized, Message: errorMessageUnauthorized},
				HTTPStatus:    http.StatusUnauthorized,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusUnauthorized, nil))
		})

		t.Run("403の場合、AccessDeniedのエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeAccessDenied, Message: errorMessageAccessDenied},
				HTTPStatus:    http.StatusForbidden,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusForbidden, nil))
		})

		t.Run("422の場合、ValidationFailedのエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeValidationFailed, Message: errorMessageValidationFailed},
				HTTPStatus:    http.StatusUnprocessableEntity,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusUnprocessableEntity, nil))
		})

		t.Run("429の場合、TooManyRequestsのエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeTooManyRequests, Message: errorMessageTooManyRequests},
				HTTPStatus:    http.StatusTooManyRequests,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusTooManyRequests, nil))
		})

		t.Run("499の場合、ClientClosedRequestのエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeClientClosedRequest, Message: errorMessageClientClosedRequest},
				HTTPStatus:    statusClientClosedRequest,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(statusClientClosedRequest, nil))
		})

		t.Run("500の場合、InternalErrorのエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeInternalError, Message: errorMessageInternalError},
				HTTPStatus:    http.StatusInternalServerError,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusInternalServerError, nil))
		})

		t.Run("501の場合、NotImplementedのエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeNotImplemented, Message: errorMessageNotImplemented},
				HTTPStatus:    http.StatusNotImplemented,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusNotImplemented, nil))
		})

		t.Run("503の場合、ServiceUnavailableのエラーが返る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeServiceUnavailable, Message: errorMessageServiceUnavailable},
				HTTPStatus:    http.StatusServiceUnavailable,
			}
			assert.Equal(t, want, NewHTTPErrorFromStatus(http.StatusServiceUnavailable, nil))
		})
	})
}

func TestHTTPErrorResponse_Error(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Internalが無い場合、HTTPステータスとメッセージが表示される", func(t *testing.T) {
			t.Parallel()
			httpError := &HTTPErrorResponse{
				HTTPStatus: http.StatusBadRequest,
				ErrorResponse: gen.ErrorResponse{
					Code:    codeBadRequest,
					Message: errorMessageBadRequest,
				},
			}

			expected := "HTTP 400: BAD_REQUEST (入力内容に誤りがあります。再度ご確認ください。)"

			assert.Equal(t, expected, httpError.Error())
		})

		t.Run("Internalがある場合、内部エラーの内容が表示される", func(t *testing.T) {
			t.Parallel()
			internalErr := errors.New("some internal error")
			httpError := &HTTPErrorResponse{
				HTTPStatus: http.StatusInternalServerError,
				ErrorResponse: gen.ErrorResponse{
					Code:    codeInternalError,
					Message: errorMessageInternalError,
				},
				Internal: internalErr,
			}

			assert.Equal(t, internalErr.Error(), httpError.Error())
		})
	})
}

func Test_newHTTPErrorFromMeta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("詳細がない場合、DetailsはnilになりHTTPStatus等が設定される", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: "E001", Message: "something wrong"},
				HTTPStatus:    400,
			}
			assert.Equal(t, want, newHTTPErrorFromMeta(httpErrorMeta{Code: "E001", Message: "something wrong", Status: 400}))
		})

		t.Run("詳細が1つある場合、Detailsにその値が入る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: "E002", Message: "one detail", Details: new([]string{"detail1"})},
				HTTPStatus:    422,
			}
			assert.Equal(t, want, newHTTPErrorFromMeta(httpErrorMeta{Code: "E002", Message: "one detail", Status: 422}, "detail1"))
		})

		t.Run("詳細が複数ある場合、Detailsに全て入る", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: "E003", Message: "many details", Details: new([]string{"d1", "d2", "d3"})},
				HTTPStatus:    500,
			}
			assert.Equal(t, want, newHTTPErrorFromMeta(httpErrorMeta{Code: "E003", Message: "many details", Status: 500}, "d1", "d2", "d3"))
		})

		t.Run("メタ情報のゼロ値を渡した場合、そのまま反映される", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{},
				HTTPStatus:    0,
			}
			assert.Equal(t, want, newHTTPErrorFromMeta(httpErrorMeta{}))
		})

		t.Run("詳細に空文字を渡した場合、Detailsポインタが生成され空文字を含む", func(t *testing.T) {
			t.Parallel()
			want := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: "E004", Message: "empty detail", Details: new([]string{""})},
				HTTPStatus:    409,
			}
			assert.Equal(t, want, newHTTPErrorFromMeta(httpErrorMeta{Code: "E004", Message: "empty detail", Status: 409}, ""))
		})
	})
}
