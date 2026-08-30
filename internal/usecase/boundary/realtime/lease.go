//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import (
	"context"
	"time"
)

// InstanceID は、serve instance の識別子です。
type InstanceID string

// InstanceLease は、serve instance の生存記録です。lock でも leader election でもなく、
// crash した instance の配送 resource を後から回収するためだけに使います（ADR-0073）。
type InstanceLease struct {
	// InstanceID は、記録の主です。
	InstanceID InstanceID
	// HeartbeatAt は、最後に生存を報告した時刻です。
	HeartbeatAt time.Time
	// ExpiresAt は、この時刻を過ぎて更新が無ければ instance を死んだと見なす期限です。
	ExpiresAt time.Time
	// CleanupOwner は、回収を引き受けた主体の識別子です。未回収なら空です。
	CleanupOwner string
	// CleanupOwnerUntil は、回収の引き受けが有効な期限です。過ぎれば他の主体が引き受け直せます。
	CleanupOwnerUntil time.Time
}

// CleanupClaim は、死んだ instance の回収を引き受ける要求です。
type CleanupClaim struct {
	// InstanceID は、回収対象です。
	InstanceID InstanceID
	// Owner は、引き受ける主体の識別子です。
	Owner string
	// ExpiredBefore は、lease の ExpiresAt がこの時刻より前であるときだけ引き受けます
	// （expiry に safety margin を足した値を呼び出し側が渡します）。
	ExpiredBefore time.Time
	// Now は、既存の引き受けが失効しているかの判定時刻です。
	Now time.Time
	// OwnerUntil は、引き受けの有効期限です。
	OwnerUntil time.Time
}

// CleanupRelease は、回収を終えた lease を閉じる要求です。
type CleanupRelease struct {
	// InstanceID は、閉じる対象です。
	InstanceID InstanceID
	// Owner は、回収を引き受けていた主体の識別子です。
	Owner string
	// ExpiredBefore は、lease の ExpiresAt がこの時刻より前であるときだけ閉じます。引き受けている間に
	// instance が heartbeat を再開していれば、その lease はもう回収対象ではないので閉じません。
	ExpiredBefore time.Time
}

// InstanceLeaseStore は、instance lease の保存境界です。失敗は apperror sentinel で返します。
type InstanceLeaseStore interface {
	// Heartbeat は、lease を作成または更新します（HeartbeatAt / ExpiresAt を書き換える）。
	Heartbeat(ctx context.Context, lease InstanceLease) error
	// Delete は、lease を削除します。無くてもエラーになりません。
	Delete(ctx context.Context, id InstanceID) error
	// ListExpired は、asOf 時点で ExpiresAt を過ぎた lease を返します。
	ListExpired(ctx context.Context, asOf time.Time) ([]InstanceLease, error)
	// AcquireCleanup は、claim の条件（期限切れ、かつ未回収か引き受けが失効）を満たすときだけ
	// 引き受けを記録して true を返します。他の主体が先に引き受けていれば false を返します。
	AcquireCleanup(ctx context.Context, claim CleanupClaim) (bool, error)
	// ReleaseCleanup は、release の条件（引き受けが owner のままで、かつ lease がまだ期限切れ）を
	// 満たすときだけ lease を削除して true を返します。引き受けが他へ移った、instance が heartbeat を
	// 再開した、lease が既に無い、のいずれでも false を返します。
	ReleaseCleanup(ctx context.Context, release CleanupRelease) (bool, error)
}
