// Package event は、問い合わせの outbox イベントの転送契約（Published Language）を持ちます。
// version 込みの種別名と JSON のフィールド名はここが所有し、ドメインは知りません。
package event

import (
	"encoding/json"
	"time"

	"go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/pkg/xerrors"
)

// TypeMessageCreated は、問い合わせメッセージ追加の outbox イベント種別（version 込み）です。
const TypeMessageCreated = "inquiry.message.created.v1"

// messageAuthor は、送り手のうち購読側へ出す部分です。主体 ID は出しません
// （会話画面は「利用者か回答者か」だけを必要とし、ID を配ると宛先の識別子が stream の外へ広がります）。
type messageAuthor struct {
	Kind string `json:"kind"`
}

// messageCreated は、inquiry.message.created.v1 の自己完結 snapshot payload です。
type messageCreated struct {
	MessageID string        `json:"messageId"`
	InquiryID string        `json:"inquiryId"`
	Author    messageAuthor `json:"author"`
	Body      string        `json:"body"`
	Sequence  int64         `json:"sequence"`
	CreatedAt string        `json:"createdAt"`
}

// BuildMessageCreated は、メッセージ集約から inquiry.message.created.v1 の payload を marshal します。
func BuildMessageCreated(m *inquirymessage.Message) ([]byte, error) {
	payload, err := json.Marshal(messageCreated{
		MessageID: m.ID().String(),
		InquiryID: m.InquiryID().String(),
		Author:    messageAuthor{Kind: m.Author().Kind().String()},
		Body:      m.Body(),
		Sequence:  m.Sequence(),
		CreatedAt: m.CreatedAt().Format(time.RFC3339Nano),
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to encode inquiry.message.created payload")
	}
	return payload, nil
}
