// Package worker は、pull-ack クラスのキューを consume する worker の seam（port）と
// broker 非依存のメッセージ封筒を定義します。engine（controller 層）と
// broker adapter（infrastructure 層）の双方が依存する境界です。
package worker

const (
	// ReservedKeyPrefix は、engine が解釈も伝播もしない adapter 専用の broker 固有値
	// （receipt handle / lease 等）を、traceparent 等の伝播対象属性と区別して隔離するための予約キー接頭辞です。
	ReservedKeyPrefix = "_"

	// AttrReceiptHandle は、broker のメッセージ識別子（SQS の receipt handle 等）を
	// Attributes に隔離するための予約キーです。adapter が設定し、Ack/Nack/Extend 時に読み戻します。
	AttrReceiptHandle = ReservedKeyPrefix + "receipt_handle"
)

// Message は、broker 非依存のメッセージ封筒です。
type Message struct {
	// ID は broker のメッセージ ID です（ログ・冪等キーのヒント）。
	ID string
	// Body はメッセージ本文です。
	Body []byte
	// Attributes は engine が解釈せず素通しする値（traceparent 等）と、
	// ReservedKeyPrefix 付きの broker 固有値（handle/lease）を運びます。
	Attributes map[string]string
	// ReceiveCount は再配送回数です（poison 検出に使用）。
	ReceiveCount int
	// PartitionKey は同一 key のメッセージを直列化するための正規化キーです（順序不要なら空）。
	// adapter が broker 値（SQS の MessageGroupId 等）を正規化して詰めます。
	PartitionKey string
}
