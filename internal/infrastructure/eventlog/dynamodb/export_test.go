package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/usecase/boundary/realtime"
)

// deleteEventForTest は、event item だけを消します。保持期間の掃除で event が消えた後の状態を
// テストが作るための口で、watermark item は残します。
func (s *store) deleteEventForTest(ctx context.Context, streamID realtime.StreamID, seq realtime.Sequence) error {
	_, err := s.c.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.table),
		Key:       key(streamID, seq),
	})
	if err != nil {
		return dynamodbclient.Normalize(err, "delete event for test")
	}

	return nil
}
