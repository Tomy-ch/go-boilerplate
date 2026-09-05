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
	// Scope は、発行するクーポンの適用範囲です。検証済みの値オブジェクトを受け取ります。
	// どの範囲を配るかは業務の判断なので、決めるのも組み立てるのも呼び出し側です。
	Scope coupon.Scope
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
	// 集約ではなく発行条件を受け取る理由と、往復数が母集団に比例しない根拠は
	// ADR-0034 (commandservice-atomicity-criterion) の Worked instances を参照。
	// 個々の Coupon は受給者を読んだあとにドメインのコンストラクタを通して組み立てます。
	IssueDiscontinuationCoupons(ctx context.Context, params IssueDiscontinuationCouponsParams) (IssueDiscontinuationCouponsResult, error)
}
