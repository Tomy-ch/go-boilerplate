// Package membership は、ユーザーの在籍と購入の進行状態にまたがる規則を表すドメインサービスです。
//
// 「在籍していないユーザーは購入できない」と「進行中の購入が残っているユーザーは退会できない」は、
// 一つの不変条件（ユーザーと進行中の購入を切り離してはならない）を購入側と退会側から見た姿です。
// ユーザー集約は購入の進行状態を知らず、購入集約は購入者の在籍を知らないため、この規則はどちらの
// 集約の自然な責務にもなりません。状態を持たず、読み込み済みの状態だけを受け取って可否を返すため、
// 集約の外に置くドメインサービスとして表現します。
//
// 判定に必要な状態の取得（ロック・問い合わせ）と、その順序・トランザクション境界は Usecase の
// 責務です。本パッケージは I/O を持たず、context.Context も受け取りません。
package membership

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrPurchaserWithdrawn は、退会済みの購入者による購入を拒否したことを表します。
	ErrPurchaserWithdrawn = xerrors.Wrap(apperror.ErrConflict, "purchaser is withdrawn")
	// ErrInProgressPurchaseExists は、進行中の購入が残っているユーザーの退会を拒否したことを表します。
	ErrInProgressPurchaseExists = xerrors.Wrap(apperror.ErrConflict, "user has in-progress purchases")
)

// EnsurePurchasable は、購入者が購入してよい状態かを判定します。
// 在籍していない（退会済みの）購入者には ErrPurchaserWithdrawn を返します。
func EnsurePurchasable(purchaser *user.User) error {
	if !purchaser.IsActive() {
		return ErrPurchaserWithdrawn
	}
	return nil
}

// EnsureWithdrawable は、ユーザーが退会してよい状態かを判定します。statuses は、そのユーザーの
// 購入が取っているステータスです（重複の有無は問いません）。進行中の購入が 1 件でも残っている
// 場合は ErrInProgressPurchaseExists を返します。
//
// 進行中かどうかの定義は購入集約が持ちます（purchase.Status.IsTerminal の否定）。
// 既に退会しているユーザーは退会の対象になり得ないため、user.ErrAlreadyDeleted を返します。
func EnsureWithdrawable(withdrawing *user.User, statuses []purchase.Status) error {
	if !withdrawing.IsActive() {
		return user.ErrAlreadyDeleted
	}
	for _, status := range statuses {
		if !status.IsTerminal() {
			return ErrInProgressPurchaseExists
		}
	}
	return nil
}
