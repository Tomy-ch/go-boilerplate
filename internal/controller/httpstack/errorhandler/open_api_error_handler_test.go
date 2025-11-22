package errorhandler

import (
	"fmt"
	"testing"

	"boilerplate-go/internal/controller/error/response"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/require"
)

func Test_normalizeOpenAPIError(t *testing.T) {
	t.Parallel()

	t.Run("RequestErrorの場合、BadRequestが返る", func(t *testing.T) {
		t.Parallel()

		reqErr := &openapi3filter.RequestError{}

		actual := normalizeOpenAPIError(reqErr)
		require.NotNil(t, actual)

		expected := response.NewHTTPErrorFromStatus(400)
		require.Equal(t, expected.Code, actual.Code)
		require.Equal(t, expected.Message, actual.Message)
		require.Equal(t, expected.HTTPStatus, actual.HTTPStatus)
		// Details は渡していないので nil
		require.Nil(t, actual.Details)
		// Internal は元のエラーが設定される
		require.Equal(t, reqErr, actual.Internal)
	})

	t.Run("SecurityRequirementsErrorの場合、Unauthorisedが返る", func(t *testing.T) {
		t.Parallel()

		secErr := &openapi3filter.SecurityRequirementsError{}

		actual := normalizeOpenAPIError(secErr)
		require.NotNil(t, actual)

		expected := response.NewHTTPErrorFromStatus(401)
		require.Equal(t, expected.Code, actual.Code)
		require.Equal(t, expected.Message, actual.Message)
		require.Equal(t, expected.HTTPStatus, actual.HTTPStatus)
		require.Nil(t, actual.Details)
		require.Equal(t, secErr, actual.Internal)
	})

	t.Run("ResponseErrorの場合、InternalServerErrorが返る", func(t *testing.T) {
		t.Parallel()

		respErr := &openapi3filter.ResponseError{}

		actual := normalizeOpenAPIError(respErr)
		require.NotNil(t, actual)

		expected := response.NewHTTPErrorFromStatus(500)
		require.Equal(t, expected.Code, actual.Code)
		require.Equal(t, expected.Message, actual.Message)
		require.Equal(t, expected.HTTPStatus, actual.HTTPStatus)
		require.Nil(t, actual.Details)
		require.Equal(t, respErr, actual.Internal)
	})

	t.Run("その他のエラーの場合、nilが返る", func(t *testing.T) {
		t.Parallel()

		actual := normalizeOpenAPIError(nil)
		require.Nil(t, actual)

		actual = normalizeOpenAPIError(fmt.Errorf("just an error"))
		require.Nil(t, actual)
	})

	t.Run("detailsを渡した場合、Detailsにセットされる", func(t *testing.T) {
		t.Parallel()

		reqErr := &openapi3filter.RequestError{}

		actual := normalizeOpenAPIError(reqErr, "d1", "d2")
		require.NotNil(t, actual)
		require.NotNil(t, actual.Details)
		require.Equal(t, []string{"d1", "d2"}, *actual.Details)
	})
}
