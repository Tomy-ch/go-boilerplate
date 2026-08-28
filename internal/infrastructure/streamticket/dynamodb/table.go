// Package dynamodb は、StreamTicket 境界（realtime.StreamTicketStore）の DynamoDB 実装を提供します。
// partition key は ticket の hash で、接続時の照合は key 1 発の強い一貫性の読み取りです。
// subject × destination での無効化は、その 2 つを複合キーにした GSI（KEYS_ONLY）で hash を引いてから削除します
// （連結した 1 つのキーにしないのは、区切り文字を含む値で別の組と衝突する余地を残さないため）。
package dynamodb

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"go-boilerplate/internal/infrastructure/dynamodbclient"
)

// item の属性名と index 名。
const (
	attrTicketHash    = "ticket_hash"
	attrSubject       = "subject"
	attrDestination   = "destination"
	attrScope         = "scope"
	attrInitialCursor = "initial_cursor"
	attrIssuedAt      = "issued_at"
	attrExpiresAt     = "expires_at"

	indexBySubjectDestination = "by_subject_destination"
)

// TableSpec は、StreamTicket table の定義を返します（one-shot の初期化と contract test が使います）。
func TableSpec(name string) dynamodbclient.TableSpec {
	return dynamodbclient.TableSpec{
		Name: name,
		Attributes: []types.AttributeDefinition{
			{AttributeName: aws.String(attrTicketHash), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(attrSubject), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(attrDestination), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(attrTicketHash), KeyType: types.KeyTypeHash},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{{
			IndexName: aws.String(indexBySubjectDestination),
			KeySchema: []types.KeySchemaElement{
				{AttributeName: aws.String(attrSubject), KeyType: types.KeyTypeHash},
				{AttributeName: aws.String(attrDestination), KeyType: types.KeyTypeRange},
			},
			Projection: &types.Projection{ProjectionType: types.ProjectionTypeKeysOnly},
		}},
		TTLAttribute: attrExpiresAt,
	}
}
