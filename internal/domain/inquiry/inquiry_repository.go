//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package inquiry

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// HistoryParams は、問い合わせのメッセージ取得条件です。
type HistoryParams struct {
	// AfterSequence は、取得を開始する位置です。nil は先頭から取得します。
	AfterSequence *int64
	// UpToSequence は、取得する上限の位置です。usecase が先に読んだ stream の現在位置で、
	// これを上限にすることで現在位置と同じ snapshot で読んだのと等価になります。
	UpToSequence int64
	// Limit は、取得する最大件数です。
	Limit int
}

// Repository は、問い合わせ集約の永続化を抽象化します。
// メッセージは集約の内側にあるため、その読み書きもこの Repository が担います。
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

	// Update は、問い合わせの更新日時を永続化します（AppendMessage が進めた値）。
	Update(ctx context.Context, inquiry *Inquiry) error

	// ListForOperator は、運営向けに問い合わせを更新日時の新しい順（updatedAt desc, id desc）で
	// keyset ページネーションして取得します。読み取り専用です。
	ListForOperator(ctx context.Context, params ListParams) ([]*Inquiry, error)

	// CreateMessage は、問い合わせへメッセージを 1 件追加します。業務 tx の中から、機構の採番の後に
	// 呼びます。問い合わせ内の位置が重複した場合は Conflict を返します
	// （採番と同一 tx で呼ぶ限り到達しない防御です）。
	CreateMessage(ctx context.Context, inquiryID uuid.UUID, message *Message) error

	// ListMessages は、問い合わせのメッセージを位置の昇順で keyset ページネーションして取得します。
	// History API の本体です。
	ListMessages(ctx context.Context, inquiryID uuid.UUID, params HistoryParams) ([]*Message, error)
}
