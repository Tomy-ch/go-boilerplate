package testkit

import (
	"regexp"
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

	assert.Regexp(t, regexp.MustCompile(`^test_event_log_[0-9a-f]{12}$`), a)
	assert.NotEqual(t, a, b, "呼び出しごとに異なる")
}

func TestDeleteOnCleanup(t *testing.T) {
	t.Parallel()

	c := NewTestClient(t)
	table := TableName(t, "cleanup")
	_, err := c.CreateTable(t.Context(), &dynamodb.CreateTableInput{
		TableName:            aws.String(table),
		AttributeDefinitions: []dynamodbtypes.AttributeDefinition{{AttributeName: aws.String("pk"), AttributeType: dynamodbtypes.ScalarAttributeTypeS}},
		KeySchema:            []dynamodbtypes.KeySchemaElement{{AttributeName: aws.String("pk"), KeyType: dynamodbtypes.KeyTypeHash}},
		BillingMode:          dynamodbtypes.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	t.Run("後片付けを登録すると terminal で table が消える", func(t *testing.T) {
		DeleteOnCleanup(t, c, table)
	})

	_, err = c.DescribeTable(t.Context(), &dynamodb.DescribeTableInput{TableName: aws.String(table)})
	require.Error(t, err, "subtest の cleanup 後には table が無い")
}
