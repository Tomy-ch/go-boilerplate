// Package instancelease は、InstanceLease 境界（realtime.InstanceLeaseStore）の実装を選ぶ唯一の場所です。
package instancelease

import (
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"go-boilerplate/internal/infrastructure/instancelease/dynamodb"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/realtime"
)

// New は、DynamoDB adapter を table に向けて構築し realtime.InstanceLeaseStore を返します。
func New(c *awsdynamodb.Client, table string, tf observability.TracerFactory) realtime.InstanceLeaseStore {
	return dynamodb.New(c, table, tf)
}
