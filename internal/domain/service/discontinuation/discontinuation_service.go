// Package discontinuation は、商品の廃番と進行中の購入にまたがる規則を表すドメインサービスです。
//
// 「進行中の購入がある商品は廃番にできない」は、商品と購入のどちらの自然な責務にもなりません。
// 商品集約は自分を含む購入を知らず、購入集約は商品が廃番にできるかを知らないためです。
// 状態を持たず、読み込み済みの状態だけを受け取って可否を返すため、集約の外に置くドメインサービスとして
// 表現します（membership と同じ形）。
//
// 判定に必要な状態の取得（ロック・問い合わせ）と、その順序・トランザクション境界は Usecase の
// 責務です。本パッケージは I/O を持たず、context.Context も受け取りません。
package discontinuation

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/pkg/xerrors"
)

// ErrInProgressPurchaseExists は、進行中の購入が残っている商品の廃番を拒否したことを表します。
var ErrInProgressPurchaseExists = xerrors.Wrap(apperror.ErrConflict, "product has in-progress purchases")

// EnsureDiscontinuable は、商品を廃番にしてよい状態かを判定します。statuses は、その商品を明細に
// 持つ購入が取っているステータスです（重複の有無は問いません）。進行中の購入が 1 件でも残っている
// 場合は ErrInProgressPurchaseExists を返します。
//
// 進行中かどうかの定義は購入集約が持ちます（purchase.Status.IsTerminal の否定）。
func EnsureDiscontinuable(statuses []purchase.Status) error {
	for _, s := range statuses {
		if !s.IsTerminal() {
			return ErrInProgressPurchaseExists
		}
	}

	return nil
}
