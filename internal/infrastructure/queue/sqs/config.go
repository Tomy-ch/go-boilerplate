// Package sqs は、worker seam（Consumer / FailureHandler）と outbox の publish 境界
// （Publisher）の AWS SQS 実装を提供します。
//
// 本パッケージの配線は、サンプル削除で外れる形に限ります（SQS の知識を本パッケージと
// それを選ぶ配線だけに閉じ込めるため。ADR-0048 の E3）。本番利用時は integrator が、受信側を WorkerModule へ
// 配線し、送出側を outbox の publish 先として選びます。詳細は README.md を参照。
package sqs

// Config は、SQS Consumer の adapter 固有設定です（engine-core の WorkerConfig とは分離）。
type Config struct {
	// QueueURL は、consume 対象キューの URL です。
	QueueURL string
	// DLQURL は、DLQ の URL です。空の場合は DLQ の滞留量を取得しません。
	DLQURL string
	// MaxMessages は、ReceiveMessage の最大取得件数です（SQS の上限は 10）。
	MaxMessages int32
	// WaitTimeSeconds は、long-poll の待機秒数です（0〜20）。
	WaitTimeSeconds int32
	// VisibilityTimeout は、受信メッセージの可視性タイムアウト秒数です。
	VisibilityTimeout int32
}
