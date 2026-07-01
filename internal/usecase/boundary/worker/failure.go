//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package worker

import "context"

// FailureHandler は、永久失敗（Permanent）メッセージの退避先 seam（dead-letter）です。
// engine は Permanent と分類したメッセージを Fail へ流してから、source を Ack で除去します。
type FailureHandler interface {
	// Fail は、永久失敗メッセージを退避先へ送ります。cause は分類のもとになったエラーです。
	Fail(ctx context.Context, m Message, cause error) error
}
