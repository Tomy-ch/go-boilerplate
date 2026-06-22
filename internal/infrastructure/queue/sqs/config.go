// Package sqs は、worker seam（Consumer / FailureHandler）の AWS SQS 参考実装です。
//
// この adapter は seam が fake 以外の 2nd impl でも成立することを示す参考実装であり、
// cmd の default 配線からは import されません（出荷バイナリに aws-sdk を載せないため。E3）。
// 本番利用する場合は integrator が WorkerModule に明示的に配線します。
//
// dead-letter の一般経路は worker.FailureHandler（本 package の DeadLetter）です。
// SQS の redrive policy（maxReceiveCount→DLQ）は IaC 側の設定であり、app は ReceiveCount の監視のみ行います。
package sqs

// Config は、SQS Consumer の adapter 固有設定です（engine-core の WorkerConfig とは分離）。
type Config struct {
	// QueueURL は、consume 対象キューの URL です。
	QueueURL string
	// MaxMessages は、ReceiveMessage の最大取得件数です（SQS の上限は 10）。
	MaxMessages int32
	// WaitTimeSeconds は、long-poll の待機秒数です（0〜20）。
	WaitTimeSeconds int32
	// VisibilityTimeout は、受信メッセージの可視性タイムアウト秒数です。
	VisibilityTimeout int32
}
