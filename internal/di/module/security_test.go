package module

import (
	"testing"
)

func Test_securityModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// パスワードハッシュ化（bcrypt hasher）の配線のみを検証する。SecurityConfig は commonDeps が供給する。
	opts := append(commonDeps(), securityModule())
	validateGraph(t, opts...)
}
