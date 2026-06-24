package exchangerate_test

import (
	"testing"

	"go-boilerplate/internal/infrastructure/external/exchangerate"

	"github.com/stretchr/testify/assert"
)

func TestNewEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サンプル既定のEndpointを返す", func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, exchangerate.NewEndpoint())
		})
	})
}
