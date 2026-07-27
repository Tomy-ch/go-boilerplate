package module

import (
	"testing"
)

func Test_objectStorageModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	opts := append(commonDeps(), objectStorageModule())
	validateGraph(t, opts...)
}

func Test_objectStorageModule(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func Test_provideObjectStorage(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
