// Package inquiry は、問い合わせドメインを定義します。利用者が運営に対して開始する一連のやり取りを
// 表し、利用者 1 人につき 1 件という制約と、最終更新時刻の単調性を不変条件として保持します。
//
// 問い合わせが持つ状態は「誰が始めたか」と「最後にいつ動いたか」だけです。close / reopen のような
// 状態遷移は持ちません。やり取りの中身は問い合わせメッセージ（internal/domain/inquirymessage）が
// 独立した集約として持ちます。
package inquiry

import (
	"time"

	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Inquiry は、問い合わせを表すドメイン集約です。
type Inquiry struct {
	id        uuid.UUID
	userID    uuid.UUID
	createdAt time.Time
	updatedAt time.Time
}

// Attributes は、問い合わせの生成と再構築に必要な属性一式です。id と userID は同じ uuid.UUID の
// ため、取り違えを型で防ぐ目的で identity 以外を構造体で受けます
// （docs/rules.md の Function Signature Rules）。
type Attributes struct {
	// UserID は、問い合わせを開始した利用者です。
	UserID uuid.UUID
	// CreatedAt は、作成日時です。生成時はゼロ値で、DB の既定値が入ります。
	CreatedAt time.Time
	// UpdatedAt は、最後にメッセージが追加された日時です。生成時はゼロ値です。
	UpdatedAt time.Time
}

// Cursor は、運営向け一覧の keyset ページネーションの位置です。
type Cursor struct {
	// UpdatedAt は、直前のページ末尾の更新日時です。
	UpdatedAt time.Time
	// ID は、更新日時が同値の行を決着させる直前のページ末尾の ID です。
	ID uuid.UUID
}

// ListParams は、運営向け一覧の取得条件です。
type ListParams struct {
	// Cursor は、取得を開始する位置です。nil は先頭ページを意味します。
	Cursor *Cursor
	// Limit は、取得する最大件数です。
	Limit int
}

// New は、問い合わせを生成します。最初の投稿時に、active な問い合わせが無い場合だけ呼ばれます。
// id が未設定なら ErrInvalidID を、UserID が未設定なら ErrInvalidUserID を返します。
func New(id uuid.UUID, attrs Attributes) (*Inquiry, error) {
	return newInquiry(id, attrs)
}

// Reconstruct は、永続化済みの問い合わせを再構築します。
// New と同じ不変条件を課します。保存済みデータのための緩和経路はありません。
// 加えて、作成日時が設定されている場合は更新日時がそれ以降であることを要求します。
func Reconstruct(id uuid.UUID, attrs Attributes) (*Inquiry, error) {
	i, err := newInquiry(id, attrs)
	if err != nil {
		return nil, err
	}
	if !attrs.CreatedAt.IsZero() && attrs.UpdatedAt.Before(attrs.CreatedAt) {
		return nil, xerrors.Wrap(ErrInvalidTime, "updatedAt must be at or after createdAt")
	}
	return i, nil
}

// newInquiry は、2 つの入口が共有する検証ゲートです。
func newInquiry(id uuid.UUID, attrs Attributes) (*Inquiry, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if attrs.UserID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidUserID, "userID is required")
	}

	return &Inquiry{
		id:        id,
		userID:    attrs.UserID,
		createdAt: attrs.CreatedAt,
		updatedAt: attrs.UpdatedAt,
	}, nil
}

// ID は、問い合わせ ID を返します。
func (i *Inquiry) ID() uuid.UUID { return i.id }

// UserID は、問い合わせを開始した利用者の ID を返します。
func (i *Inquiry) UserID() uuid.UUID { return i.userID }

// CreatedAt は、作成日時を返します。生成直後の集約ではゼロ値で、再構築時に設定されます。
func (i *Inquiry) CreatedAt() time.Time { return i.createdAt }

// UpdatedAt は、最後にメッセージが追加された日時を返します。生成直後の集約ではゼロ値で、
// 再構築時に設定されます。
func (i *Inquiry) UpdatedAt() time.Time { return i.updatedAt }

// Touch は、メッセージが追加されたことを記録し、更新日時を now へ進めます。
// now が現在の更新日時より前なら ErrInvalidTime を返します（更新日時は単調に進みます）。
func (i *Inquiry) Touch(now time.Time) error {
	if now.Before(i.updatedAt) {
		return xerrors.Wrap(ErrInvalidTime, "now must be at or after updatedAt")
	}
	i.updatedAt = now
	return nil
}
