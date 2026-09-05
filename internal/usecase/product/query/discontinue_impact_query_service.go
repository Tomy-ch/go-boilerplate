//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package query

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// DiscontinueImpactReadModel は、商品を廃番にしたときの影響の見積もりです。
type DiscontinueImpactReadModel struct {
	// AffectedCartCount は、対象商品の明細を持つカートの件数です。ゲストのカートも含みます。
	AffectedCartCount int64
	// AffectedUserCount は、クーポンの受給対象になる確定済みユーザーの数です。
	// ゲストのカートと退会済みユーザーを除くため AffectedCartCount 以下になります。
	AffectedUserCount int64
	// InProgressPurchaseCount は、対象商品を含む進行中の購入の件数です。1 以上なら廃番は拒まれます。
	InProgressPurchaseCount int64
}

// DiscontinueImpactQueryService は、廃番の影響の集約跨ぎ read 投影を提供する QueryService です。
// カート・ユーザー・購入をまたぐ件数の投影であり、どの集約も再構成しないため読み取り側に置きます
// （ADR-0032 (lightweight-cqrs)）。
type DiscontinueImpactQueryService interface {
	// EstimateDiscontinueImpact は、商品を廃番にした場合の影響を件数で返します。
	//
	// **行をロックしません。** 返した値は返した瞬間から古くなり、実行時の件数と一致する保証はありません。
	// 押す前に規模を見せるための読み取りであり、可否の判定そのものは実行時のトランザクションが持ちます。
	//
	// 各件数の母集団は CommandService 側の書き込みと 1 対 1 で対応します。片方だけを変えると、
	// 見積もりと実行が食い違って押す前に見せた数字の意味が失われます。
	EstimateDiscontinueImpact(ctx context.Context, productID uuid.UUID) (DiscontinueImpactReadModel, error)
}
