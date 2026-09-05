//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package coupon

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// Repository は、クーポン集約の永続化を抽象化します。
//
// 廃番に伴う一括発行はここに持ちません。発行対象が述語でしか決まらず件数に上限が無いため、
// 集約を 1 件ずつ構築して書く形に分解できないためです。その書き込みは CommandService が担います
// （判定基準は ADR-0034、実例は docs/spec/usecase/product.md の廃番）。
type Repository interface {
	// CountByScopeTargetProductID は、指定した商品を適用範囲の対象とするクーポンの発行枚数を返します。
	// 廃番を再実行したときに、新たな発行を伴わずに実績を返すために用います。
	CountByScopeTargetProductID(ctx context.Context, productID uuid.UUID) (int, error)
}
