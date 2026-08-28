package testkit

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestClient(t *testing.T) {
	t.Parallel()

	c := NewTestClient(t)
	require.NotNil(t, c)
	assert.Equal(t, "http://localhost:8000", aws.ToString(c.Options().BaseEndpoint), "既定は DynamoDB Local")
}

func TestTableName(t *testing.T) {
	t.Parallel()

	a := TableName(t, "event_log")
	b := TableName(t, "event_log")

	assert.Regexp(t, `^test_event_log_[0-9a-f]{12}$`, a)
	assert.NotEqual(t, a, b, "呼び出しごとに異なる")
}

func TestDeleteOnCleanup(t *testing.T) {
	t.Parallel()

	c := NewTestClient(t)
	table := TableName(t, "cleanup")
	_, err := c.CreateTable(t.Context(), &dynamodb.CreateTableInput{
		TableName: aws.String(table),
		AttributeDefinitions: []dynamodbtypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: dynamodbtypes.ScalarAttributeTypeS},
		},
		KeySchema:   []dynamodbtypes.KeySchemaElement{{AttributeName: aws.String("pk"), KeyType: dynamodbtypes.KeyTypeHash}},
		BillingMode: dynamodbtypes.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	// cleanup は LIFO で走るので、後から登録する DeleteOnCleanup の削除が先に済んでからここで確かめる。
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
		defer cancel()

		_, err := c.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)})
		assert.Error(t, err, "DeleteOnCleanup の後には table が無い")
	})
	DeleteOnCleanup(t, c, table)
}
