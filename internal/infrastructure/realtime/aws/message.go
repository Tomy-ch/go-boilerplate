package aws

import (
	"encoding/json"
	"strconv"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/apperror"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// AttrKind は、wakeup と失効通知を 1 つの topic で見分ける MessageAttribute の名前です。
// 本文の形は ADR-0073 が固定しているので、種別は本文ではなく属性で運びます（RawMessageDelivery でも透過する）。
const AttrKind = "type"

// stringAttr は、SNS の String 属性の DataType です。
const stringAttr = "String"

// wakeupBody は、wakeup の本文です（eventId / streamId / sequence のみ。sequence は 10 進文字列）。
type wakeupBody struct {
	EventID  string `json:"eventId"`
	StreamID string `json:"streamId"`
	Sequence string `json:"sequence"`
}

// revocationBody は、失効通知の本文です。
type revocationBody struct {
	Subject     string `json:"subject"`
	Destination string `json:"destination"`
}

// encodeWakeup は、wakeup の本文と種別属性を返します。
func encodeWakeup(w rt.Wakeup) (string, map[string]snstypes.MessageAttributeValue, error) {
	b, err := json.Marshal(wakeupBody{EventID: w.EventID, StreamID: string(w.StreamID), Sequence: w.Sequence.String()})
	if err != nil {
		return "", nil, xerrors.Wrap(apperror.ErrInternal, "encode wakeup: "+err.Error())
	}

	return string(b), kindAttribute(rt.KindWakeup), nil
}

// encodeRevocation は、失効通知の本文と種別属性を返します。
func encodeRevocation(r rt.Revocation) (string, map[string]snstypes.MessageAttributeValue, error) {
	b, err := json.Marshal(revocationBody{Subject: r.Subject, Destination: string(r.Destination)})
	if err != nil {
		return "", nil, xerrors.Wrap(apperror.ErrInternal, "encode revocation: "+err.Error())
	}

	return string(b), kindAttribute(rt.KindRevocation), nil
}

func kindAttribute(kind rt.NotificationKind) map[string]snstypes.MessageAttributeValue {
	return map[string]snstypes.MessageAttributeValue{
		AttrKind: {DataType: awssdk.String(stringAttr), StringValue: awssdk.String(string(kind))},
	}
}

// decodeNotification は、受信した message を Notification へ復元します。種別が無い・読めない message は
// Kind を空にして返し、受け取る側が削除できるよう Receipt だけは必ず載せます（残すと再配送され続けるため）。
func decodeNotification(m sqstypes.Message) rt.Notification {
	n := rt.Notification{Receipt: awssdk.ToString(m.ReceiptHandle)}

	attr, ok := m.MessageAttributes[AttrKind]
	if !ok {
		return n
	}

	body := []byte(awssdk.ToString(m.Body))
	switch kind := rt.NotificationKind(awssdk.ToString(attr.StringValue)); kind {
	case rt.KindWakeup:
		var w wakeupBody
		if json.Unmarshal(body, &w) != nil {
			return n
		}

		seq, err := strconv.ParseInt(w.Sequence, 10, 64)
		if err != nil {
			return n
		}

		n.Kind = kind
		n.Wakeup = rt.Wakeup{EventID: w.EventID, StreamID: rt.StreamID(w.StreamID), Sequence: rt.Sequence(seq)}
	case rt.KindRevocation:
		var r revocationBody
		if json.Unmarshal(body, &r) != nil {
			return n
		}

		n.Kind = kind
		n.Revocation = rt.Revocation{Subject: r.Subject, Destination: rt.StreamID(r.Destination)}
	}

	return n
}
