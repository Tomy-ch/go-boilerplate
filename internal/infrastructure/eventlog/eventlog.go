// Package eventlog は、EventLog 境界（realtime.EventLogStore）の実装を選ぶ唯一の場所です。
// 背後の substrate を差し替える場合に書き換えるのはこのパッケージだけで、DI はここを通ります。
package eventlog

import (
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"go-boilerplate/internal/infrastructure/eventlog/dynamodb"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/realtime"
)

// New は、DynamoDB adapter を table に向けて構築し realtime.EventLogStore を返します。
// クライアントは dynamodbclient.New で組み立てたものを DI が渡します（3 store で 1 つを共有する）。
func New(c *awsdynamodb.Client, table string, tf observability.TracerFactory) realtime.EventLogStore {
	return dynamodb.New(c, table, tf)
}
