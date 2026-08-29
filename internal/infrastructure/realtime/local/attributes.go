// Package local は、SNS / SQS emulator（GoAWS）が wire 互換でないと実測された呼び出しにだけ当てる
// compatibility implementation です（ADR-0073: 第 2 の protocol adapter は作らない）。
// 対象は instance queue の属性の 1 点で、fan-out・lifecycle API・種別属性は native の実装をそのまま使います。
package local

import (
	realtimeaws "go-boilerplate/internal/infrastructure/realtime/aws"
)

// queueAttributes は、emulator が受け付ける属性だけを返します。
type queueAttributes struct{}

// NewQueueAttributes は、long polling と visibility timeout だけを設定する AttributesBuilder を返します。
// GoAWS v0.5.4 に対する `make realtime-smoke` の実測で、Policy は InvalidParameterValue で拒否され（G4 / G4b）、
// RedrivePolicy は受理後に受信した message を削除できなくなり（G15）、SqsManagedSseEnabled と KmsMasterKeyId は
// 受理されるが保存されない（G16 / G17）ため、この 4 つを間引きます。production の実装は間引きません。
func NewQueueAttributes() realtimeaws.AttributesBuilder {
	return queueAttributes{}
}

func (queueAttributes) Build(string) (map[string]string, error) {
	return realtimeaws.TimingAttributes(), nil
}
