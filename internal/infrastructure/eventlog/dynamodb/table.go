// Package dynamodb は、EventLog 境界（realtime.EventLogStore）の DynamoDB 実装を提供します。
// partition key は stream、sort key は sequence で、1 stream の event が sequence 順に並びます。
// 期限切れの掃除は TTL に任せますが、replay できるかの判定（保持期間）は usecase 側の時計で行います。
package dynamodb

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"go-boilerplate/internal/infrastructure/dynamodbclient"
)

// item の属性名。
const (
	attrStreamID      = "stream_id"
	attrSequence      = "sequence"
	attrEventID       = "event_id"
	attrType          = "event_type"
	attrOccurredAt    = "occurred_at"
	attrSchemaVersion = "schema_version"
	attrPayload       = "payload"
	attrExpiresAt     = "expires_at"
)

// TableSpec は、EventLog table の定義を返します（one-shot の初期化と contract test が使います）。
func TableSpec(name string) dynamodbclient.TableSpec {
	return dynamodbclient.TableSpec{
		Name: name,
		Attributes: []types.AttributeDefinition{
			{AttributeName: aws.String(attrStreamID), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(attrSequence), AttributeType: types.ScalarAttributeTypeN},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(attrStreamID), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String(attrSequence), KeyType: types.KeyTypeRange},
		},
		TTLAttribute: attrExpiresAt,
	}
}
