package purchase

import (
	"fmt"

	"go-boilerplate/pkg/xerrors"
)

// 既知のステータスの業務キー。**到達順序を意味しません**
// （完了=5 / キャンセル=6 より支払い済み=7 のほうが大きい）。
const (
	statusCodeUnprocessed = 1
	statusCodeCompleted   = 5
	statusCodeCanceled    = 6
	statusCodePaid        = 7
	statusCodeShipped     = 8
	statusCodeDelivered   = 9
)

// 既知のステータス。ステータスの UUID はドメインに焼き込まず、永続化時に code から解決します
// （seed との二重管理を避けるため）。
var (
	// StatusUnprocessed は、購入作成直後に設定される「未処理」です。
	StatusUnprocessed = Status{code: statusCodeUnprocessed, name: "unprocessed"}
	// StatusCompleted は、購入完了です。完了後はキャンセルできません。
	StatusCompleted = Status{code: statusCodeCompleted, name: "completed"}
	// StatusCanceled は、購入キャンセルです。発送前にのみ到達します。
	StatusCanceled = Status{code: statusCodeCanceled, name: "canceled"}
	// StatusPaid は、支払い済みです。未払い相当からのみ到達します。
	StatusPaid = Status{code: statusCodePaid, name: "paid"}
	// StatusShipped は、発送済みです。支払い済みからのみ到達します。
	StatusShipped = Status{code: statusCodeShipped, name: "shipped"}
	// StatusDelivered は、配達済みです。発送済みからのみ到達する終端状態です。
	StatusDelivered = Status{code: statusCodeDelivered, name: "delivered"}
)

// Status は、購入のステータスを表す値オブジェクトです。
//
// 内側に持つ code は永続化と外部公開のための業務キーであり、到達順序を意味しません。
// したがって遷移の可否や終端性を code の大小で判定してはならず、必ず本型のメソッドを通します。
type Status struct {
	code int
	name string
}

// allStatuses は、既知のステータス一覧です。code からの解決に用います。
func allStatuses() []Status {
	return []Status{
		StatusUnprocessed, StatusCompleted, StatusCanceled,
		StatusPaid, StatusShipped, StatusDelivered,
	}
}

// NewStatus は、永続化されている code からステータスを解決します。
// 既知でない code は ErrInvalidStatusID を返します（永続化状態の破損を再構築時に弾くため）。
func NewStatus(code int) (Status, error) {
	for _, s := range allStatuses() {
		if s.code == code {
			return s, nil
		}
	}
	return Status{}, xerrors.Wrap(ErrInvalidStatusID, fmt.Sprintf("unknown status code: %d", code))
}

// Code は、永続化と外部公開に用いる業務キーを返します。到達順序の比較には使えません。
func (s Status) Code() int { return s.code }

// Name は、ステータスの名前を返します。外部へ状態を伝えるときは code ではなくこちらを用います。
func (s Status) Name() string { return s.name }

// IsZero は、未設定のステータスかどうかを返します。
func (s Status) IsZero() bool { return s.code == 0 }

// IsTerminal は、そこから他の状態へ遷移しない終端状態かどうかを返します。
// 終端でないステータスの購入は進行中として扱います。
func (s Status) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusCanceled, StatusDelivered:
		return true
	default:
		return false
	}
}

// CanTransitionTo は、next へ遷移できるかを返します。ステータスだけで決まる可否を判定するもので、
// 集約の状態（発送記録の有無など）に依存する条件は集約側が併せて検証します。
func (s Status) CanTransitionTo(next Status) bool {
	if s.IsTerminal() || s == next {
		return false
	}
	switch next {
	case StatusCanceled:
		// キャンセルは進行中からのみ。発送済みからのキャンセル不可は集約が発送記録で併せて弾く。
		return true
	case StatusPaid:
		// 支払いは未払い相当からのみ。
		return s == StatusUnprocessed
	case StatusShipped:
		return s == StatusPaid
	case StatusDelivered:
		return s == StatusShipped
	default:
		return false
	}
}
