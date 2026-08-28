// Package streamticket は、StreamTicket 境界（realtime.StreamTicketStore）の実装を選ぶ唯一の場所です。
package streamticket

import (
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"go-boilerplate/internal/infrastructure/streamticket/dynamodb"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/realtime"
)

// New は、DynamoDB adapter を table に向けて構築し realtime.StreamTicketStore を返します。
func New(c *awsdynamodb.Client, table string, tf observability.TracerFactory) realtime.StreamTicketStore {
	return dynamodb.New(c, table, tf)
}
