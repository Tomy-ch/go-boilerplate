package errorhandler

import (
	"errors"
	"testing"

	"go-boilerplate/internal/controller/error/response"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_normalizeOpenAPIError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RequestErrorの場合、BadRequestが返る", func(t *testing.T) {
			t.Parallel()

			reqErr := &openapi3filter.RequestError{}

			actual := normalizeOpenAPIError(reqErr)
			require.NotNil(t, actual)

			expected := response.NewHTTPErrorFromStatus(400, nil)
			assert.Equal(t, expected.Code, actual.Code)
			assert.Equal(t, expected.Message, actual.Message)
			assert.Equal(t, expected.HTTPStatus, actual.HTTPStatus)
			assert.Nil(t, actual.Details)
			assert.Equal(t, reqErr, actual.Internal)
		})

		t.Run("SecurityRequirementsErrorの場合、Unauthorizedが返る", func(t *testing.T) {
			t.Parallel()

			secErr := &openapi3filter.SecurityRequirementsError{}

			actual := normalizeOpenAPIError(secErr)
			require.NotNil(t, actual)

			expected := response.NewHTTPErrorFromStatus(401, nil)
			assert.Equal(t, expected.Code, actual.Code)
			assert.Equal(t, expected.Message, actual.Message)
			assert.Equal(t, expected.HTTPStatus, actual.HTTPStatus)
			assert.Nil(t, actual.Details)
			assert.Equal(t, secErr, actual.Internal)
		})

		t.Run("ResponseErrorの場合、InternalServerErrorが返る", func(t *testing.T) {
			t.Parallel()

			respErr := &openapi3filter.ResponseError{}

			actual := normalizeOpenAPIError(respErr)
			require.NotNil(t, actual)

			expected := response.NewHTTPErrorFromStatus(500, nil)
			assert.Equal(t, expected.Code, actual.Code)
			assert.Equal(t, expected.Message, actual.Message)
			assert.Equal(t, expected.HTTPStatus, actual.HTTPStatus)
			assert.Nil(t, actual.Details)
			assert.Equal(t, respErr, actual.Internal)
		})

		t.Run("detailsを渡した場合、Detailsにセットされる", func(t *testing.T) {
			t.Parallel()

			reqErr := &openapi3filter.RequestError{}

			actual := normalizeOpenAPIError(reqErr, "d1", "d2")
			require.NotNil(t, actual)
			require.NotNil(t, actual.Details)
			assert.Equal(t, []string{"d1", "d2"}, *actual.Details)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilを渡すとnilが返る", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, normalizeOpenAPIError(nil))
		})

		t.Run("OpenAPI由来でないエラーはnilが返る", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, normalizeOpenAPIError(errors.New("just an error")))
		})
	})
}
