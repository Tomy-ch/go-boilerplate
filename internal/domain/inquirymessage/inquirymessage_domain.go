// Package inquirymessage は、問い合わせメッセージドメインを定義します。問い合わせの中で一方が
// 相手に送る 1 通を表し、作成後の編集も取り消しもありません（append-only）。
//
// メッセージは問い合わせを経由せず一覧・ページングされるため、問い合わせの sub-entity ではなく
// 独立した集約です。問い合わせへは識別子だけで参照します。
//
// sequence は「その問い合わせの何通目か」を表し、配送順序の基準として Realtime Delivery が
// 要求する値です。採番は機構が行い、この層は正の整数であることだけを検証します——問い合わせ内での
// 一意性と単調増加は集約 1 件では判定できません。
package inquirymessage

import (
	"fmt"
	"time"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Message は、問い合わせメッセージを表すドメイン集約です。
type Message struct {
	id        uuid.UUID
	inquiryID uuid.UUID
	author    Author
	body      string
	sequence  int64
	createdAt time.Time
}

// Attributes は、メッセージの生成と再構築に必要な属性一式です。id と inquiryID は同じ uuid.UUID の
// ため、取り違えを型で防ぐ目的で identity 以外を構造体で受けます
// （docs/rules.md の Function Signature Rules）。
type Attributes struct {
	// InquiryID は、所属する問い合わせです。集約跨ぎの参照は識別子のみです。
	InquiryID uuid.UUID
	// Author は、送り手です。
	Author Author
	// Body は、本文です。
	Body string
	// Sequence は、問い合わせ内の位置（1 起算）です。usecase が機構の採番結果を渡します。
	Sequence int64
	// CreatedAt は、作成日時です。生成時はゼロ値で、DB の既定値が入ります。
	CreatedAt time.Time
}

// New は、メッセージを生成します。投稿時と回答時に usecase が呼びます。
func New(id uuid.UUID, attrs Attributes) (*Message, error) {
	return newMessage(id, attrs)
}

// Reconstruct は、永続化済みのメッセージを再構築します。
// New と同じ不変条件を課します。保存済みデータのための緩和経路はありません。
func Reconstruct(id uuid.UUID, attrs Attributes) (*Message, error) {
	return newMessage(id, attrs)
}

// newMessage は、2 つの入口が共有する検証ゲートです。
func newMessage(id uuid.UUID, attrs Attributes) (*Message, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if attrs.InquiryID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidInquiryID, "inquiryID is required")
	}
	if err := validateAuthor(attrs.Author.kind, attrs.Author.subjectID); err != nil {
		return nil, err
	}
	if err := validateBody(attrs.Body); err != nil {
		return nil, err
	}
	if attrs.Sequence < 1 {
		return nil, xerrors.Wrap(
			ErrInvalidSequence, fmt.Sprintf("sequence must be 1 or greater, got %d", attrs.Sequence),
		)
	}

	return &Message{
		id:        id,
		inquiryID: attrs.InquiryID,
		author:    attrs.Author,
		body:      attrs.Body,
		sequence:  attrs.Sequence,
		createdAt: attrs.CreatedAt,
	}, nil
}

// validateBody は、本文が空でなく上限文字数以下であることを検証します。
// 長さは rune 数で数えます（バイト数で数えるとマルチバイト文字の本文を早すぎる時点で拒否します）。
func validateBody(body string) error {
	if body == "" {
		return xerrors.Wrap(ErrEmptyBody, "body is required")
	}
	if ok, msg := stringkit.ValidateInRange(body, minBodyLength, maxBodyLength); !ok {
		return xerrors.Wrap(ErrBodyTooLong, msg)
	}
	return nil
}

// ID は、メッセージ ID を返します。
func (m *Message) ID() uuid.UUID { return m.id }

// InquiryID は、所属する問い合わせの ID を返します。
func (m *Message) InquiryID() uuid.UUID { return m.inquiryID }

// Author は、送り手を返します。
func (m *Message) Author() Author { return m.author }

// Body は、本文を返します。
func (m *Message) Body() string { return m.body }

// Sequence は、問い合わせ内の位置を返します。
func (m *Message) Sequence() int64 { return m.sequence }

// CreatedAt は、作成日時を返します。生成直後の集約ではゼロ値で、再構築時に設定されます。
func (m *Message) CreatedAt() time.Time { return m.createdAt }

// IsFrom は、送り手が指定の種別・主体であるかを返します。
// 履歴の表示や event の送り手の導出に使う純粋な述語で、状態を変えません。
func (m *Message) IsFrom(kind AuthorKind, subjectID uuid.UUID) bool {
	return m.author.kind == kind && m.author.subjectID == subjectID
}
