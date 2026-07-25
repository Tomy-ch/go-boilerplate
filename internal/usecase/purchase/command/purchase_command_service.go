//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package command は、購入の書き込み操作（CommandService）のインターフェースを定義します（ADR-0027）。
// 実装は infra 層に置き、渡された ctx のトランザクションに参加します。outbox 発行は含めません（usecase 責務）。
package command

import (
	"context"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/pkg/uuid"
)

// CommandService は、購入に伴う複数集約（products 在庫 / purchases / purchase_details）への
// 原子的な書き込みを定義します。金額計算・売り越し判定・スナップショットはドメインが持ち、本サービスは
// 「決定済みの書き込みの実行」のみを担います（Repository の write 側対称物）。
type CommandService interface {
	// LockProducts は、指定商品を ID 昇順に悲観ロック（SELECT FOR UPDATE）し、価格・在庫を返します。
	// ロック順序を固定することでデッドロックを避けます。
	LockProducts(ctx context.Context, productIDs []uuid.UUID) ([]purchase.LockedProduct, error)
	// CreatePurchase は、在庫減算・purchases INSERT・purchase_details INSERT を、渡された ctx の
	// トランザクション内で原子的に実行します。在庫減算は防御的に売り越しを弾きます。
	CreatePurchase(ctx context.Context, p *purchase.Purchase) error
	// LockPurchase は、購入行を悲観ロック（SELECT FOR UPDATE）して明細込みで再構築し返します。
	// キャンセルの状態遷移の競合（同一購入への並行キャンセル）を購入行ロックで直列化します。
	// 存在しない場合は NotFound を返します。
	LockPurchase(ctx context.Context, id uuid.UUID) (*purchase.Purchase, error)
	// CancelPurchase は、キャンセルに伴う在庫復元（明細分の加算）と購入の状態更新（status_id / canceled_at）を、
	// 渡された ctx のトランザクション内で原子的に実行します。在庫加算は相対更新で売り越しを生みません。
	CancelPurchase(ctx context.Context, p *purchase.Purchase) error
}
