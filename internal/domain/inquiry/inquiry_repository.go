//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package inquiry

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// Repository は、問い合わせ集約の永続化を抽象化します。
type Repository interface {
	// FindByID は、問い合わせを 1 件取得します。存在しない場合は NotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*Inquiry, error)

	// FindActiveByUserID は、利用者の active な問い合わせを取得します。
	// 利用者 1 人につき高々 1 件です。存在しない場合は NotFound を返します。
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*Inquiry, error)

	// Create は、問い合わせを新規登録します。
	// 同じ利用者の active な問い合わせが既にある場合は Conflict を返します。
	// この Conflict は同一 tx の中で読み直して解決できないため、呼び出し側は tx をやり直します
	// （docs/spec/inquiry/usecase.md の AppendMessage）。
	Create(ctx context.Context, inquiry *Inquiry) error

	// Touch は、最後にメッセージが追加された日時を更新します（Inquiry.Touch の永続化）。
	Touch(ctx context.Context, id uuid.UUID, now time.Time) error

	// ListForOperator は、運営向けに問い合わせを更新日時の新しい順（updatedAt desc, id desc）で
	// keyset ページネーションして取得します。読み取り専用です。
	ListForOperator(ctx context.Context, params ListParams) ([]*Inquiry, error)
}
