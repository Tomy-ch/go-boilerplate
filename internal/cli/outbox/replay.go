package outbox

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// ReplayFunc は、messageID（任意）を対象に dead→pending 復帰を実行し戻した件数を返す関数の型です。
type ReplayFunc func(ctx context.Context, messageID *uuid.UUID) (int64, error)

// RunReplayWith は、rawMessageID を解釈して replay を実行し、戻した件数を返します。
// rawMessageID が空なら全 dead 行、指定時は当該 message_id のみを対象とします。
func RunReplayWith(ctx context.Context, rawMessageID string, replay ReplayFunc) (int64, error) {
	var id *uuid.UUID
	if rawMessageID != "" {
		parsed, err := uuid.Parse(rawMessageID)
		if err != nil {
			return 0, err
		}
		id = &parsed
	}
	return replay(ctx, id)
}
