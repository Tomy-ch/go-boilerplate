package aws

import (
	"encoding/json"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// policyVersion は、IAM policy document の version です。
const policyVersion = "2012-10-17"

// queuePolicy は、instance queue の access policy です。topic からの送信だけを許します。
type queuePolicy struct {
	Version   string            `json:"Version"`   //nolint:tagliatelle // IAM policy document のフィールド名
	Statement []policyStatement `json:"Statement"` //nolint:tagliatelle // 同上
}

type policyStatement struct {
	Sid       string            `json:"Sid"`       //nolint:tagliatelle // IAM policy document のフィールド名
	Effect    string            `json:"Effect"`    //nolint:tagliatelle // 同上
	Principal map[string]string `json:"Principal"` //nolint:tagliatelle // 同上
	Action    string            `json:"Action"`    //nolint:tagliatelle // 同上
	Resource  string            `json:"Resource"`  //nolint:tagliatelle // 同上
	Condition map[string]any    `json:"Condition"` //nolint:tagliatelle // 同上
}

// queuePolicyDocument は、queueARN への sqs:SendMessage を topicARN の SNS からだけ許す policy を返します。
// これが無い queue は同じ account の任意の principal から書けてしまうため、production では必ず設定します。
func queuePolicyDocument(queueARN, topicARN string) (string, error) {
	doc, err := json.Marshal(queuePolicy{
		Version: policyVersion,
		Statement: []policyStatement{{
			Sid:       "AllowRealtimeTopic",
			Effect:    "Allow",
			Principal: map[string]string{"Service": "sns.amazonaws.com"},
			Action:    "sqs:SendMessage",
			Resource:  queueARN,
			Condition: map[string]any{"ArnEquals": map[string]string{"aws:SourceArn": topicARN}},
		}},
	})
	if err != nil {
		return "", xerrors.Wrap(apperror.ErrInternal, "encode queue policy: "+err.Error())
	}

	return string(doc), nil
}
