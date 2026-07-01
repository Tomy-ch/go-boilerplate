package validator

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetValidator(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OpenAPIバリデータを生成する", func(t *testing.T) {
			t.Parallel()

			validator, err := GetValidator()
			require.NoError(t, err)
			assert.IsType(t, &openapi3.T{}, validator)
		})
	})
}
