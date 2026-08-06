// Package dispatch は、発送待ちの購入をまとめて発送してよい組へ分ける規則を表すドメインサービスです。
//
// 「この購入は発送可能か」は 1 件の購入自身の状態だけで決まるため Purchase のメソッド（IsShippable）
// ですが、「これらのうちどれとどれを 1 便にまとめてよいか」は集合についての問いで、ある 1 件についての
// 答えが集合に他のどれが含まれるかに依存します。どの Purchase 1 件のメソッドにもなり得ないため、
// 購入集約の自然な責務にはなりません。状態を持たず、読み込み済みの状態だけを受け取って組を返すため、
// 集約の外に置くドメインサービスとして表現します。
//
// 語るのは購入集約だけで、import するのも internal/domain/purchase だけです。この置き場が集約の
// import を許すのは許可であって要求ではありません（internal/domain/README.md）。
//
// 対象の取得（絞り込み・件数の上限）と、取得した行が実際に発送可能かの検証、および
// トランザクション境界は Usecase の責務です。本パッケージは I/O を持たず、context.Context も
// 受け取りません。
package dispatch

import (
	"cmp"
	"slices"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/pkg/uuid"
)

// GroupForDispatch は、発送待ちの購入を、まとめて発送してよい組に分けます。
// 同一の購入者宛てであることを軸とし、同じ購入者の購入は 1 つの組にまとまります。
//
// 受け取るのは発送可能な購入の集合であることを前提とし、本関数は発送可能性を検証しません
// （その定義は Purchase.IsShippable が持ち、取得した行との突き合わせは Usecase が行います）。
// 同梱可能期間などの追加条件は現時点では持ちません。
//
// 結果は入力の並び順に依存せず一意に定まります。組の中は注文日時の古い順（同時刻は購入 ID の昇順）、
// 組同士はその組の最も古い購入の同じ順序で並びます。購入が 1 件だけの購入者もその 1 件からなる組に
// なります。purchases が空の場合は空を返します。
func GroupForDispatch(purchases purchase.Purchases) []purchase.Purchases {
	byPurchaser := make(map[uuid.UUID]purchase.Purchases, len(purchases))
	for _, p := range purchases {
		byPurchaser[p.UserID()] = append(byPurchaser[p.UserID()], p)
	}

	groups := make([]purchase.Purchases, 0, len(byPurchaser))
	for _, group := range byPurchaser {
		slices.SortFunc(group, compareDispatchOrder)
		groups = append(groups, group)
	}
	slices.SortFunc(groups, func(a, b purchase.Purchases) int {
		return compareDispatchOrder(a[0], b[0])
	})

	return groups
}

// compareDispatchOrder は、購入 a と b の発送順を比較します。注文日時の古い順、同時刻は購入 ID の
// 昇順です。同時刻の並びを ID で決めるのは、順序を一意にするためです。
func compareDispatchOrder(a, b *purchase.Purchase) int {
	if !a.OrderedAt().Equal(b.OrderedAt()) {
		return a.OrderedAt().Compare(b.OrderedAt())
	}
	return cmp.Compare(a.ID().String(), b.ID().String())
}
