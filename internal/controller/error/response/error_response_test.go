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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("detailsがある場合、detailsが表示されるエラー構造体が返る", func(t *testing.T) {
			t.Parallel()

			httpStatus := http.StatusBadRequest
			err := fmt.Errorf("bad request error: %w", apperror.ErrInvalidArgument)
			details := []string{"invalid input", "missing field"}

			expected := &HTTPErrorResponse{}
			expected.HTTPStatus = httpStatus
			expected.Code = codeBadRequest
			expected.Message = errorMessageBadRequest
			expected.Details = ptr.To(details)
			expected.Internal = err

			actual := NewHTTPErrorFromAppError(err, details...)

			assert.Equal(t, expected, actual)
		})

		t.Run("detailsがない場合、detailsは表示されないエラー構造体が返る", func(t *testing.T) {
			t.Parallel()
			httpStatus := http.StatusInternalServerError
			err := fmt.Errorf("internal server error: %w", apperror.ErrInternal)

			expected := &HTTPErrorResponse{}
			expected.HTTPStatus = httpStatus
			expected.Code = codeInternalError
			expected.Message = errorMessageInternalError
			expected.Internal = err

			actual := NewHTTPErrorFromAppError(err)

			assert.Equal(t, expected, actual)
		})
	})
}

func TestNewHTTPErrorFromStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知のステータスコードの場合、対応するエラーが返る", func(t *testing.T) {
			t.Parallel()

			httpStatus := http.StatusNotFound

			actual := NewHTTPErrorFromStatus(httpStatus)

			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    codeNotFound,
					Message: errorMessageNotFound,
					Details: nil,
				},
				HTTPStatus: httpStatus,
			}

			assert.Equal(t, expected, actual)
		})

		t.Run("詳細が渡された場合、Detailsにセットされる", func(t *testing.T) {
			t.Parallel()

			httpStatus := http.StatusConflict
			actual := NewHTTPErrorFromStatus(httpStatus, "conflict-1")

			ds := []string{"conflict-1"}
			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    codeResourceConflict,
					Message: errorMessageResourceConflict,
					Details: &ds,
				},
				HTTPStatus: httpStatus,
			}

			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のステータスコードの場合、内部エラーとして扱われる", func(t *testing.T) {
			t.Parallel()

			httpStatus := 999

			actual := NewHTTPErrorFromStatus(httpStatus)

			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    codeInternalError,
					Message: errorMessageInternalError,
					Details: nil,
				},
				HTTPStatus: http.StatusInternalServerError,
			}

			assert.Equal(t, expected, actual)
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

		assert.Equal(t, expected, actual)
	})
}

func TestHTTPErrorResponse_Error(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("HTTPステータスとメッセージが表示される", func(t *testing.T) {
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

			assert.Equal(t, expected, actual)
		})

		t.Run("内部エラーがある場合、内部エラーの内容も表示される", func(t *testing.T) {
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

			expected := internalErr.Error()
			actual := httpError.Error()

			assert.Equal(t, expected, actual)
		})
	})
}

func Test_newHTTPErrorFromMeta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("詳細がない場合、DetailsはnilになりHTTPStatus等が設定される", func(t *testing.T) {
			t.Parallel()

			meta := httpErrorMeta{
				Code:    "E001",
				Message: "something wrong",
				Status:  400,
			}

			actual := newHTTPErrorFromMeta(meta)

			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    "E001",
					Message: "something wrong",
					Details: nil,
				},
				HTTPStatus: 400,
			}

			assert.Equal(t, expected, actual)
		})

		t.Run("詳細が1つある場合、Detailsにその値が入る", func(t *testing.T) {
			t.Parallel()

			meta := httpErrorMeta{
				Code:    "E002",
				Message: "one detail",
				Status:  422,
			}

			actual := newHTTPErrorFromMeta(meta, "detail1")

			ds := []string{"detail1"}
			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    "E002",
					Message: "one detail",
					Details: &ds,
				},
				HTTPStatus: 422,
			}

			assert.Equal(t, expected, actual)
		})

		t.Run("詳細が複数ある場合、Detailsに全て入る", func(t *testing.T) {
			t.Parallel()

			meta := httpErrorMeta{
				Code:    "E003",
				Message: "many details",
				Status:  500,
			}

			actual := newHTTPErrorFromMeta(meta, "d1", "d2", "d3")

			ds := []string{"d1", "d2", "d3"}
			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    "E003",
					Message: "many details",
					Details: &ds,
				},
				HTTPStatus: 500,
			}

			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メタ情報のゼロ値を渡す場合、そのまま反映される", func(t *testing.T) {
			t.Parallel()

			meta := httpErrorMeta{}

			actual := newHTTPErrorFromMeta(meta)

			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    "",
					Message: "",
					Details: nil,
				},
				HTTPStatus: 0,
			}

			assert.Equal(t, expected, actual)
		})

		t.Run("詳細に空文字を渡した場合、Detailsポインタが生成され空文字を含む", func(t *testing.T) {
			t.Parallel()

			meta := httpErrorMeta{
				Code:    "E004",
				Message: "empty detail",
				Status:  409,
			}

			actual := newHTTPErrorFromMeta(meta, "")

			ds := []string{""}
			expected := &HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:    "E004",
					Message: "empty detail",
					Details: &ds,
				},
				HTTPStatus: 409,
			}

			assert.Equal(t, expected, actual)
		})
	})
}
