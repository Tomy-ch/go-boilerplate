// Package conv は、controller 層のリクエスト/レスポンス双方の境界型変換ヘルパーを提供します。
// OpenAPI 生成型 ↔ ドメイン型の変換や、usecase の DTO をレスポンス型へ倒す変換をここへ集約し、
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
