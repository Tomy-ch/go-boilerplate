// Package conv は、controller 層で OpenAPI 生成型をドメイン型へ変換する境界ヘルパーを提供します。
// OpenAPI 生成型を import するのは controller 層のみであり、本パッケージ経由に集約することで
// 変換の用途を境界に限定します（usecase / domain からは利用できません）。
package conv

import (
	"fmt"

	"go-boilerplate/pkg/uuid"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// UUID は、OpenAPI 生成の UUID（path / query パラメータ）をドメインの uuid.UUID へ変換します。
// 値は echo のバインド時に UUID 形式が検証済みのため必ず変換可能で、エラーは返しません。
// 万一変換できない場合は、到達してはならない不変条件違反（致命的バグ）として panic します。
func UUID(p openapi_types.UUID) uuid.UUID {
	return mustParseUUID(p.String())
}

// mustParseUUID は、UUID 文字列をドメイン UUID へ変換し、不正な場合は panic します。
// 到達した場合は呼び出し側の前提（検証済みの値のみ渡す）が壊れていることを意味します。
func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("conv: unexpected invalid UUID string %q: %v", s, err))
	}
	return id
}
