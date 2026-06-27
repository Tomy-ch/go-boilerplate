package httpclient

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientDo_AllowRetryWithoutKey_Guard は、公開 API（NewRequest / WithRetry）では構築不能な
// 「allowRetry あり・idempotencyKey なし」という不正状態に対する Do の防御的ガードを検証します。
//
// 非公開フィールドを直接組み立てられるパッケージ内からのみ到達可能な分岐のため、外部テストでは
// なくこのパッケージ内テストで担保します。
func TestClientDo_AllowRetryWithoutKey_Guard(t *testing.T) {
	t.Parallel()

	// ガードは Do の最初の文で、ネットワーク・registry 等に触れる前に返るため、ゼロ値 client で足りる。
	c := &client{}
	req := &Request{
		downstream: "retry",
		method:     MethodPost(),
		url:        "http://example.com",
		allowRetry: true,
	}

	resp, err := c.Do(context.Background(), req)

	require.ErrorIs(t, err, apperror.ErrInvalidArgument)
	assert.Nil(t, resp)
}
