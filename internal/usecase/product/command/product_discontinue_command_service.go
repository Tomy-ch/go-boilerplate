//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package command は、廃番の書き込み操作（CommandService）のインターフェースを定義します
// （ADR-0032 (lightweight-cqrs)）。実装は infra 層に置き、渡された ctx のトランザクションに参加します。
//
// 所在が usecase 層なのは、CommandService がトランザクションの道具であり、所有者はトランザクションを
// 開く側だからです。パッケージ名 command はワークフローの名であって集約の名ではありません。
package command

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/coupon"
	"go-boilerplate/pkg/uuid"
)

// IssueDiscontinuationCouponsParams は、廃番の代替クーポンを一括発行するための入力です。
// ExpiresAt と IssuedAt が同型のため構造体で受けます（docs/rules.md の Function Signature Rules）。
type IssueDiscontinuationCouponsParams struct {
	// ProductID は、廃番の対象です。受給者はこの商品を明細に持つカートから決まります。
	ProductID uuid.UUID
	// CategoryID は、発行するクーポンの適用範囲が指す商品カテゴリです。
	// 廃番商品自身を範囲にすると買えない商品にしか使えないため、そのカテゴリを範囲にします。
	CategoryID uuid.UUID
	// Discount は、発行するクーポンの値引きです。全員に同じ条件で配ります。
	Discount coupon.Discount
	// ExpiresAt は、発行するクーポンの有効期限です。
	ExpiresAt time.Time
	// IssuedAt は、発行日時です。
	IssuedAt time.Time
}

// IssueDiscontinuationCouponsResult は、一括発行の結果です。
type IssueDiscontinuationCouponsResult struct {
	// AffectedCartCount は、対象商品の明細を持っていたカートの件数です。ゲストのカートも含みます。
	AffectedCartCount int64
	// AffectedUserCount は、受給対象になった確定済みユーザーの数です。
	AffectedUserCount int64
	// IssuedCouponCount は、実際に発行した枚数です。受給者 1 人につき 1 枚のため AffectedUserCount と一致します。
	IssuedCouponCount int64
}

// CommandService は、廃番に伴うクーポンの一括発行を定義します。
//
// 載せてよい書き込みの基準と、強制する条件がドメイン不変条件からの導出でなければならない規律は
// ADR-0034 (commandservice-atomicity-criterion) の Eligibility / Derivation を参照。
type CommandService interface {
	// IssueDiscontinuationCoupons は、params.ProductID の明細を持つカートの所有者のうち退会していない
	// ユーザーへ、同一条件のクーポンを 1 枚ずつ発行します。渡された ctx のトランザクション内で実行します。
	//
	// 受給者は述語（cart_items への結合と退会の除外）でしか決まらず、件数に上限もないため、
	// 呼び出し側が集約を組み立てて渡すことはできません。そのため引数は「決まった集約」ではなく
	// 発行条件のテンプレートで、個々の Coupon はこのメソッドの中で採番されます
	// （data-access-pattern.md §6 の shape rule が想定する「手元の集約をバラして渡す」形とは別物です）。
	//
	// 往復は受給者の取得と挿入の 2 回で、発行枚数に比例して増えません。
	IssueDiscontinuationCoupons(ctx context.Context, params IssueDiscontinuationCouponsParams) (IssueDiscontinuationCouponsResult, error)
}
