package inquiry

import (
	"fmt"
	"time"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Message は、問い合わせの中で一方が相手に送る 1 通を表す sub-entity です。
// 作成後の編集も取り消しもありません（append-only）。
//
// sequence は「その問い合わせの何通目か」を表し、配送順序の基準として Realtime Delivery が
// 要求する値です。採番は機構が行い、この層は正の整数であることだけを検証します——問い合わせ内での
// 一意性と単調増加は 1 通だけでは判定できません。
type Message struct {
	id        uuid.UUID
	author    Author
	body      string
	sequence  int64
	createdAt time.Time
}

// MessageAttributes は、メッセージの組み立てに必要な属性一式です。
type MessageAttributes struct {
	// Author は、送り手です。
	Author Author
	// Body は、本文です。
	Body string
	// Sequence は、問い合わせ内の位置（1 起算）です。usecase が機構の採番結果を渡します。
	Sequence int64
	// CreatedAt は、作成日時です。追加時はゼロ値で、DB の既定値が入ります。
	CreatedAt time.Time
}

// ReconstructMessage は、永続化済みのメッセージを再構築します。
// AppendMessage と同じ不変条件を課します。保存済みデータのための緩和経路はありません。
func ReconstructMessage(id uuid.UUID, attrs MessageAttributes) (*Message, error) {
	return newMessage(id, attrs)
}

// newMessage は、追加と再構築が共有する検証ゲートです。
func newMessage(id uuid.UUID, attrs MessageAttributes) (*Message, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidMessageID, "id is required")
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
		author:    attrs.Author,
		body:      attrs.Body,
		sequence:  attrs.Sequence,
		createdAt: attrs.CreatedAt,
	}, nil
}

// validateBody は、本文が空でなく maxBodyLength（rune 数）以下であることを検証します。
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

// Author は、送り手を返します。
func (m *Message) Author() Author { return m.author }

// Body は、本文を返します。
func (m *Message) Body() string { return m.body }

// Sequence は、問い合わせ内の位置を返します。
func (m *Message) Sequence() int64 { return m.sequence }

// CreatedAt は、作成日時を返します。追加直後のメッセージではゼロ値で、再構築時に設定されます。
func (m *Message) CreatedAt() time.Time { return m.createdAt }

// IsFrom は、送り手が指定の種別・主体であるかを返します。
func (m *Message) IsFrom(kind AuthorKind, subjectID uuid.UUID) bool {
	return m.author.kind == kind && m.author.subjectID == subjectID
}
