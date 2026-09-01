//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package inquirymessage

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

// Repository は、問い合わせメッセージ集約の永続化を抽象化します。
type Repository interface {
	// Create は、メッセージを 1 件追加します。業務 tx の中から、機構の採番の後に呼びます。
	// 問い合わせ内の位置が重複した場合は Conflict を返します
	// （採番と同一 tx で呼ぶ限り到達しない防御です）。
	Create(ctx context.Context, message *Message) error

	// ListByInquiry は、問い合わせのメッセージを位置の昇順で keyset ページネーションして
	// 取得します。History API の本体です。
	ListByInquiry(ctx context.Context, inquiryID uuid.UUID, params HistoryParams) ([]*Message, error)
}
