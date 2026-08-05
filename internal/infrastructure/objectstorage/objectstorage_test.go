package objectstorage_test

import (
	"testing"

	"go-boilerplate/internal/config"
	objectstorage "go-boilerplate/internal/infrastructure/objectstorage"
	"go-boilerplate/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("configとTracerFactoryからStorageを生成する", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, config.Load())
			cfg, err := config.New()
			require.NoError(t, err)

			s := objectstorage.New(
				config.NewObjectStorageConfig(cfg),
				observability.NewDisabledOutboundHTTPClient(true),
				observability.NewNoopTracerFactory(t),
			)

			assert.NotNil(t, s)
		})
	})
}
