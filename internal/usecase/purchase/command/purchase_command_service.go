//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package command は、購入の書き込み操作（CommandService）のインターフェースを定義します（ADR-0031 (lightweight-cqrs)）。
// 実装は infra 層に置き、渡された ctx のトランザクションに参加します。outbox 発行は含めません（usecase 責務）。
//
// 所在が usecase 層なのは、CommandService がトランザクションの道具であり、所有者はトランザクションを
// 開く側だからです。パッケージ名 command はワークフローの名であって集約の名ではありません。この配置を
// 動かす判断は docs/adr/0031-lightweight-cqrs.md § Command Service の更新を伴います。
package command

import (
	"context"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/pkg/uuid"
)

// CommandService は、購入に伴う複数集約（商品在庫 / 購入 / 購入明細）への
// 原子的な書き込みを定義します。金額計算・売り越し判定・スナップショットはドメインが持ち、本サービスは
// 「決定済みの書き込みの実行」のみを担います（QueryService の write 側対称物）。
//
// 載せてよい書き込みの基準と、強制する条件がドメイン不変条件からの導出でなければならない規律は
// ADR-0033 (commandservice-atomicity-criterion) の Eligibility / Derivation を参照。
type CommandService interface {
	// CreatePurchase は、在庫の減算・購入の作成・明細の作成を、渡された ctx のトランザクション内で
	// 原子的に実行します。在庫減算は防御的に売り越しを弾きます。
	CreatePurchase(ctx context.Context, p *purchase.Purchase) error
	// LockPurchase は、対象の購入を悲観ロックして明細込みで再構築し返します。
	// キャンセルの状態遷移の競合（同一購入への並行キャンセル）をこのロックで直列化します。
	// 存在しない場合は NotFound を返します。
	LockPurchase(ctx context.Context, id uuid.UUID) (*purchase.Purchase, error)
	// CancelPurchase は、キャンセルに伴う在庫復元（明細分の加算）と購入の状態遷移（→ キャンセル）を、
	// 渡された ctx のトランザクション内で原子的に実行します。在庫加算は相対更新で売り越しを生みません。
	CancelPurchase(ctx context.Context, p *purchase.Purchase) error
}
