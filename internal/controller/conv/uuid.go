// Package conv は、controller 層で OpenAPI 生成型をドメイン型へ変換する境界ヘルパーを提供します。
// OpenAPI 生成型を import するのは controller 層のみであり、本パッケージ経由に集約することで
// 変換の用途を境界に限定します（usecase / domain からは利用できません）。
package conv

import (
	"go-boilerplate/pkg/uuid"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// UUID は、OpenAPI 生成の UUID（path / query パラメータ）をドメインの uuid.UUID へ変換します。
// openapi_types.UUID は検証済みの 16 バイト UUID 値型であり、変換は無条件に成功します。
func UUID(p openapi_types.UUID) uuid.UUID {
	return uuid.FromPrimitive(p)
}

// UUIDPtr は、任意指定（nullable）の UUID クエリパラメータをドメインの *uuid.UUID へ変換します。
// nil の場合はそのまま nil を返し、指定された場合のみ変換します。
func UUIDPtr(p *openapi_types.UUID) *uuid.UUID {
	if p == nil {
		return nil
	}
	id := uuid.FromPrimitive(*p)
	return &id
}
