package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/objectstorage"
	"go-boilerplate/internal/observability"
	objectstoragebd "go-boilerplate/internal/usecase/boundary/objectstorage"
)

func Test_objectStorageModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	opts := append(commonDeps(), objectStorageModule())
	validateGraph(t, opts...)
}

func Test_objectStorageModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("オブジェクトストレージ境界の Storage を提供する", func(t *testing.T) {
			t.Parallel()

			var storage objectstoragebd.Storage

			validateGraph(t, append(commonDeps(), objectStorageModule(), fx.Populate(&storage))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では Storage が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var storage objectstoragebd.Storage

			opts := append(commonDeps(), fx.Populate(&storage), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}

func Test_provideObjectStorage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定とTracerFactoryからStorage実装を構築する", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewObjectStorageConfig(config.MockConfigForTest(t))
			tf := observability.NewNoopTracerFactory(t)

			outbound := observability.NewDisabledOutboundHTTPClient(true)
			got := provideObjectStorage(cfg, outbound, tf)

			// 実装の差し替え（別 adapter や decorator の混入）を型で固定する。
			assert.IsType(t, objectstorage.New(cfg, outbound, tf), got)
		})
	})
}
