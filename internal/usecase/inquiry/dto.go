package inquiry

import (
	"time"

	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
)

// AppendMessageParams は、利用者の投稿の入力 DTO です。
type AppendMessageParams struct {
	// UserID は、認証済みの内部ユーザー ID です。
	UserID uuid.UUID
	// Subject は、Realtime Delivery へ渡す feature 非依存の principal 識別子です。
	Subject string
	// Body は、本文です。
	Body string
}

// HistoryParams は、利用者の履歴取得の入力 DTO です。
type HistoryParams struct {
	// UserID は、認証済みの内部ユーザー ID です。
	UserID uuid.UUID
	// AfterSequence は、取得を開始する位置です。nil は先頭から取得します。
	AfterSequence *int64
	// First は、要求された取得件数です。nil や範囲外はユースケースのポリシーで正規化されます。
	First *int
}

// IssueStreamTicketParams は、利用者の会話 stream 用 ticket 発行の入力 DTO です。
type IssueStreamTicketParams struct {
	// UserID は、認証済みの内部ユーザー ID です。
	UserID uuid.UUID
	// Subject は、Realtime Delivery へ渡す principal 識別子です。
	Subject string
}

// ListInquiriesParams は、運営の一覧取得の入力 DTO です。
type ListInquiriesParams struct {
	// Cursor は、取得位置と件数です。keyset の境界は不透明キー列として運ばれます
	// （購入フィードと同じ機構: internal/usecase/tools/paging）。
	Cursor *paging.Cursor
}

// OperatorHistoryParams は、運営の履歴取得の入力 DTO です。
type OperatorHistoryParams struct {
	// InquiryID は、対象の問い合わせです。
	InquiryID uuid.UUID
	// AfterSequence は、取得を開始する位置です。nil は先頭から取得します。
	AfterSequence *int64
	// First は、要求された取得件数です。nil や範囲外はユースケースのポリシーで正規化されます。
	First *int
}

// ReplyParams は、運営の回答の入力 DTO です。
type ReplyParams struct {
	// InquiryID は、対象の問い合わせです。
	InquiryID uuid.UUID
	// OperatorID は、回答者の内部ユーザー ID です。
	OperatorID uuid.UUID
	// Body は、本文です。
	Body string
}

// MessageView は、メッセージ 1 通の出力 DTO です。
type MessageView struct {
	ID         uuid.UUID
	InquiryID  uuid.UUID
	AuthorKind string
	Body       string
	Sequence   int64
	CreatedAt  time.Time
}

// HistoryView は、履歴 1 ページの出力 DTO です。
type HistoryView struct {
	// InquiryID は、対象の問い合わせです。問い合わせが無い場合はゼロ値です。
	InquiryID uuid.UUID
	// Messages は、位置の昇順に並んだメッセージです。StreamCursor 以下の位置だけを含みます。
	Messages []MessageView
	// NextAfterSequence は、次ページの開始位置です。次ページが無ければ nil です。
	NextAfterSequence *int64
	// StreamCursor は、問い合わせ stream の現在位置です。
	// client がこれを接続時の開始位置に使うと、履歴の取得と購読の間の event を取りこぼしません。
	StreamCursor int64
}

// TicketView は、発行した ticket の出力 DTO です。
type TicketView struct {
	// Ticket は、生値です。応答本文にのみ載せ、log / trace へは出しません。
	Ticket string
	// StreamID は、接続先の stream です。
	StreamID string
	// ExpiresAt は、この ticket で新しい接続を始められる期限です。
	ExpiresAt time.Time
}

// InquirySummaryView は、運営向け一覧の 1 件分の出力 DTO です。
type InquirySummaryView struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// InquiryListView は、運営向け一覧 1 ページの出力 DTO です。
type InquiryListView struct {
	Items []InquirySummaryView
	// NextCursor は、次ページ取得用の不透明カーソルです。次ページが無ければ nil です。
	NextCursor *string
}
