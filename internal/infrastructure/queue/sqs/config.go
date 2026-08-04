// Package sqs は、worker seam（Consumer / FailureHandler）の AWS SQS 実装を提供します。
//
// 本パッケージの配線は、サンプル削除で外れる形に限ります（削除後の結合をサンプル追加前と
// 同一に保つため。ADR-0106 の E3'）。本番利用時は integrator が WorkerModule に配線します。
// 詳細は README.md を参照。
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
