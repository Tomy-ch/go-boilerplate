package response

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/error/response/gen"
	"go-boilerplate/pkg/ptr"
)

func TestNewHTTPErrorFromAppError(t *testing.T) {
	t.Parallel()

	t.Run("detailsがある場合、detailsが表示されるエラー構造体が返る", func(t *testing.T) {
		t.Parallel()

		err := fmt.Errorf("bad request error: %w", apperror.ErrInvalidArgument)
		details := []string{"invalid input", "missing field"}

		expected := &HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:    codeBadRequest,
				Message: errorMessageBadRequest,
				Details: ptr.To(details),
			},
			HTTPStatus: http.StatusBadRequest,
			Internal:   err,
		}

		assert.Equal(t, expected, NewHTTPErrorFromAppError(err, details...))
	})

	t.Run("detailsがない場合、detailsは表示されないエラー構造体が返る", func(t *testing.T) {
		t.Parallel()

		err := fmt.Errorf("internal server error: %w", apperror.ErrInternal)

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
}

func TestNewHTTPErrorFromStatus(t *testing.T) {
	t.Parallel()

	internalErr := fmt.Errorf("boom")

	cases := []struct {
		name    string
		status  int
		err     error
		details []string
		want    *HTTPErrorResponse
	}{
		{
			name:   "既知のステータスコードの場合、対応するエラーが返る",
			status: http.StatusNotFound,
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeNotFound, Message: errorMessageNotFound},
				HTTPStatus:    http.StatusNotFound,
			},
		},
		{
			name:    "詳細が渡された場合、Detailsにセットされる",
			status:  http.StatusConflict,
			details: []string{"conflict-1"},
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    codeResourceConflict,
					Message: errorMessageResourceConflict,
					Details: ptr.To([]string{"conflict-1"}),
				},
				HTTPStatus: http.StatusConflict,
			},
		},
		{
			name:   "errを渡した場合、Internalに格納される",
			status: http.StatusBadRequest,
			err:    internalErr,
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeBadRequest, Message: errorMessageBadRequest},
				HTTPStatus:    http.StatusBadRequest,
				Internal:      internalErr,
			},
		},
		{
			name:   "未知のステータスコードの場合、内部エラーとして扱われる",
			status: 999,
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: codeInternalError, Message: errorMessageInternalError},
				HTTPStatus:    http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, NewHTTPErrorFromStatus(tt.status, tt.err, tt.details...))
		})
	}
}

func TestNewInternalErrorResponse(t *testing.T) {
	t.Parallel()

	t.Run("内部サーバーエラーのエラーレスポンスが生成される", func(t *testing.T) {
		t.Parallel()

		expected := &HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:    codeInternalError,
				Message: errorMessageInternalError,
			},
			HTTPStatus: http.StatusInternalServerError,
		}

		assert.Equal(t, expected, NewInternalErrorResponse())
	})
}

func TestHTTPErrorResponse_Error(t *testing.T) {
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

		expected := fmt.Sprintf("HTTP %d: %s (%s)", http.StatusBadRequest, codeBadRequest, errorMessageBadRequest)

		assert.Equal(t, expected, httpError.Error())
	})

	t.Run("Internalがある場合、内部エラーの内容が表示される", func(t *testing.T) {
		t.Parallel()
		internalErr := fmt.Errorf("some internal error")
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
}

func Test_newHTTPErrorFromMeta(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		meta    httpErrorMeta
		details []string
		want    *HTTPErrorResponse
	}{
		{
			name: "詳細がない場合、DetailsはnilになりHTTPStatus等が設定される",
			meta: httpErrorMeta{Code: "E001", Message: "something wrong", Status: 400},
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: "E001", Message: "something wrong"},
				HTTPStatus:    400,
			},
		},
		{
			name:    "詳細が1つある場合、Detailsにその値が入る",
			meta:    httpErrorMeta{Code: "E002", Message: "one detail", Status: 422},
			details: []string{"detail1"},
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: "E002", Message: "one detail", Details: ptr.To([]string{"detail1"})},
				HTTPStatus:    422,
			},
		},
		{
			name:    "詳細が複数ある場合、Detailsに全て入る",
			meta:    httpErrorMeta{Code: "E003", Message: "many details", Status: 500},
			details: []string{"d1", "d2", "d3"},
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: "E003", Message: "many details", Details: ptr.To([]string{"d1", "d2", "d3"})},
				HTTPStatus:    500,
			},
		},
		{
			name: "メタ情報のゼロ値を渡した場合、そのまま反映される",
			meta: httpErrorMeta{},
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{},
				HTTPStatus:    0,
			},
		},
		{
			name:    "詳細に空文字を渡した場合、Detailsポインタが生成され空文字を含む",
			meta:    httpErrorMeta{Code: "E004", Message: "empty detail", Status: 409},
			details: []string{""},
			want: &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{Code: "E004", Message: "empty detail", Details: ptr.To([]string{""})},
				HTTPStatus:    409,
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, newHTTPErrorFromMeta(tt.meta, tt.details...))
		})
	}
}
