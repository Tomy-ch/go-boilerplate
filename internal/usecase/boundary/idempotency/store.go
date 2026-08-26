//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package idempotency は、冪等性キーの永続化境界（Store）を定義します。
package idempotency

import (
	"context"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	// StatusClaimed は、claim 済み（保存前）の状態です。
	StatusClaimed Status = "claimed"
	// StatusCompleted は、完了し結果を保存済みの状態です。
	StatusCompleted Status = "completed"
)

// ErrLockTimeout は、並行 claim のロック待ちタイムアウトを表します（usecase 側で 409 へマップ）。
var ErrLockTimeout = xerrors.New("idempotency: claim lock timeout")

// Status は、冪等性キーの状態です。
type Status string

// Record は、replay / 409 / 422 判定に必要な保存済み状態です。ResponseStatus / ResponsePayload は completed まで nil。
type Record struct {
	Status          Status
	ResponseStatus  *int32
	ResponsePayload []byte
	Fingerprint     []byte
}

// ClaimParams は、Claim の入力です。
type ClaimParams struct {
	Scope       string
	Key         string
	Method      string
	Path        string
	Fingerprint []byte
	ExpiresAt   time.Time
}

// CompleteParams は、Complete の入力です。
type CompleteParams struct {
	Scope           string
	Key             string
	ResponseStatus  int32
	ResponsePayload []byte
}

// Store は、冪等性キーの永続化境界です。すべて scope 必須（id 単独 lookup を持たない＝越境防止）。
type Store interface {
	// Claim は、claimed のエントリを作ります。新規に作れたら claimed=true、既存キーがあれば false を返します。
	// ロック待ちタイムアウト時は ErrLockTimeout を返します。
	// 業務トランザクション内から呼び出すこと（ロック待ちの上限がトランザクションスコープのため）。
	Claim(ctx context.Context, p ClaimParams) (claimed bool, err error)
	// Get は、(scope, key) の保存済み状態を返します（存在しなければ nil, nil）。
	Get(ctx context.Context, scope, key string) (*Record, error)
	// Complete は、claimed → completed へ遷移し結果を保存します。
	Complete(ctx context.Context, p CompleteParams) error
	// DeleteExpired は、cutoff より古いエントリを limit 件まで削除し、削除件数を返します。
	DeleteExpired(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
}
