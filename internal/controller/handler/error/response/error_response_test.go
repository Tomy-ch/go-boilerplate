package errorresponse

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/error/response/gen"
	"boilerplate-go/pkg/ptr"

	"github.com/stretchr/testify/require"
)

func TestNewErrorResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("detailsがある場合、detailsが表示されるエラー構造体が返る", func(t *testing.T) {
			t.Parallel()

			httpStatus := http.StatusBadRequest
			err := errors.New("bad request error")
			details := []string{"invalid input", "missing field"}

			expected := &HTTPErrorResponse{}
			expected.HTTPStatus = httpStatus
			expected.Code = codeBadRequest
			expected.Message = errorMessageBadRequest
			expected.Details = ptr.To(details)
			expected.Internal = err

			actual := New(httpStatus, err, details...)

			require.Equal(t, expected, actual)
		})

		t.Run("detailsがない場合、detailsは表示されないエラー構造体が返る", func(t *testing.T) {
			t.Parallel()
			httpStatus := http.StatusInternalServerError
			err := errors.New("internal server error")

			expected := &HTTPErrorResponse{}
			expected.HTTPStatus = httpStatus
			expected.Code = codeInternalError
			expected.Message = errorMessageInternalError
			expected.Internal = err

			actual := New(httpStatus, err)

			require.Equal(t, expected, actual)
		})
	})
}

func TestNewInternalErrorResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系/内部サーバーエラーのエラーレスポンスが生成される", func(t *testing.T) {
		t.Parallel()

		expected := &HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:    codeInternalError,
				Message: errorMessageInternalError,
			},
			HTTPStatus: http.StatusInternalServerError,
		}

		actual := NewInternalErrorResponse()

		require.Equal(t, expected, actual)
	})
}

func TestHTTPErrorResponse_Error(t *testing.T) {
	t.Parallel()
	t.Run("正常系/HTTPステータスとメッセージが表示される", func(t *testing.T) {
		t.Parallel()
		httpError := &HTTPErrorResponse{
			HTTPStatus: http.StatusBadRequest,
			ErrorResponse: gen.ErrorResponse{
				Code:    codeBadRequest,
				Message: errorMessageBadRequest,
			},
		}

		expected := fmt.Sprintf(
			"HTTP %d: %s (%s)", http.StatusBadRequest, codeBadRequest, errorMessageBadRequest,
		)

		actual := httpError.Error()

		require.Equal(t, expected, actual)
	})
}
