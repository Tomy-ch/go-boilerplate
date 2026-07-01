//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package worker

import "context"

// Handler は、メッセージに対する業務処理です。冪等であることが契約です（at-least-once 前提）。
type Handler interface {
	// Handle は、1 件のメッセージを処理します。返すエラーの分類は apperror のセンチネルで表明します
	// （ErrRetryable / ErrPermanent / ErrFatal。未分類は engine が Retryable 既定で扱います）。
	Handle(ctx context.Context, m Message) error
}
