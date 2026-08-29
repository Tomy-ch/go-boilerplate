package aws

import (
	"encoding/json"
	"strconv"

	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// instance queue に設定する固定値。deployment で変える理由が無いので config ではなく code に置きます。
const (
	// visibilityTimeout は、受信した通知を他の受信から隠す秒数です。consumer は受信後すぐ削除するので短くて足ります。
	visibilityTimeout = 30
	// receiveWaitSeconds は、long polling の待ち秒数（SQS の上限）です。空 poll の往復を減らします。
	receiveWaitSeconds = 20
	// redriveMaxReceiveCount は、この回数受信しても削除されなかった通知を DLQ へ移す閾値です。
	redriveMaxReceiveCount = 5
)

// QueueAttributes は、instance queue に設定する属性の集合を組み立てる境界です。
// production は全属性（policy / redrive / 暗号化 / timings）を返し、emulator 向けの実装は
// emulator が受け付けない属性を間引きます。間引くのは実装側の責務で、production 側で失敗を握り潰しません。
type QueueAttributes interface {
	// Build は、queueARN の queue に設定する属性を返します。空なら何も設定しません。
	Build(queueARN string) (map[string]string, error)
}

// queueAttributes は、production の全属性です。
type queueAttributes struct {
	topicARN string
	dlqARN   string
}

// NewQueueAttributes は、topicARN からの送信だけを許す policy、dlqARN への redrive（空なら付けない）、
// SQS 管理の暗号化、long polling と visibility timeout を設定する QueueAttributes を返します。
func NewQueueAttributes(topicARN, dlqARN string) QueueAttributes {
	return &queueAttributes{topicARN: topicARN, dlqARN: dlqARN}
}

func (a *queueAttributes) Build(queueARN string) (map[string]string, error) {
	attrs := TimingAttributes()

	policy, err := queuePolicyDocument(queueARN, a.topicARN)
	if err != nil {
		return nil, err
	}

	attrs[string(sqstypes.QueueAttributeNamePolicy)] = policy
	attrs[string(sqstypes.QueueAttributeNameSqsManagedSseEnabled)] = "true"

	if a.dlqARN != "" {
		redrive, err := redrivePolicyDocument(a.dlqARN)
		if err != nil {
			return nil, err
		}

		attrs[string(sqstypes.QueueAttributeNameRedrivePolicy)] = redrive
	}

	return attrs, nil
}

// TimingAttributes は、long polling と visibility timeout の属性を返します。emulator 向けの実装も
// この 2 つは設定します（受信の振る舞いを production と揃えるため）。
func TimingAttributes() map[string]string {
	return map[string]string{
		string(sqstypes.QueueAttributeNameVisibilityTimeout):             strconv.Itoa(visibilityTimeout),
		string(sqstypes.QueueAttributeNameReceiveMessageWaitTimeSeconds): strconv.Itoa(receiveWaitSeconds),
	}
}

// redrivePolicy は、SQS の RedrivePolicy 属性の JSON です。
type redrivePolicy struct {
	DeadLetterTargetArn string `json:"deadLetterTargetArn"`
	MaxReceiveCount     string `json:"maxReceiveCount"`
}

func redrivePolicyDocument(dlqARN string) (string, error) {
	doc, err := json.Marshal(redrivePolicy{DeadLetterTargetArn: dlqARN, MaxReceiveCount: strconv.Itoa(redriveMaxReceiveCount)})
	if err != nil {
		return "", xerrors.Wrap(apperror.ErrInternal, "encode redrive policy: "+err.Error())
	}

	return string(doc), nil
}
