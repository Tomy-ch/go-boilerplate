// Package testuuid は、テスト用の UUID 生成ヘルパーを提供します。
package testuuid

import (
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
)

// RequestUUID は、リクエスト用の openapi UUID（生成型）を1件生成します。
func RequestUUID(tb testing.TB) types.UUID {
	tb.Helper()

	id, err := uuid.New()
	require.NoError(tb, err)

	return id.ToPrimitive()
}
