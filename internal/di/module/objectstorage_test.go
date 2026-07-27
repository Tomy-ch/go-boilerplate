package module

import (
	"testing"
)

func Test_objectStorageModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	opts := append(commonDeps(), objectStorageModule())
	validateGraph(t, opts...)
}
