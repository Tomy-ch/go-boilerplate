// Package dynamodb は、InstanceLease 境界（realtime.InstanceLeaseStore）の DynamoDB 実装を提供します。
// partition key は instance の識別子です。TTL は設定しません — lease が消えると crash した instance の
// resource を回収する手がかりも消えるので、期限切れの lease は回収側が読んでから削除します。
package dynamodb

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"go-boilerplate/internal/infrastructure/dynamodbclient"
)

// item の属性名。
const (
	attrInstanceID        = "instance_id"
	attrHeartbeatAt       = "heartbeat_at"
	attrExpiresAt         = "expires_at"
	attrCleanupOwner      = "cleanup_owner"
	attrCleanupOwnerUntil = "cleanup_owner_until"
)

// TableSpec は、InstanceLease table の定義を返します（one-shot の初期化と contract test が使います）。
func TableSpec(name string) dynamodbclient.TableSpec {
	return dynamodbclient.TableSpec{
		Name: name,
		Attributes: []types.AttributeDefinition{
			{AttributeName: aws.String(attrInstanceID), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(attrInstanceID), KeyType: types.KeyTypeHash},
		},
	}
}
