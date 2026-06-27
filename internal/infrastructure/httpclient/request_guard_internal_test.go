package httpclient

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientDo_AllowRetryWithoutKey_Guard は、WithRetry("") や非公開フィールド直構築で到達し得る
// 「allowRetry あり・idempotencyKey なし」という不正状態に対する Do の防御的ガードを検証します。
//
// 型では空 key を排除できないため、Do 実行時のガードで担保します。
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

// TestClientDo_EmptyMethod_Guard は、ゼロ値 Method{}（buildRequest が空文字を net/http へ渡すと
// 暗黙に GET 昇格し isRetrySafe と不整合になる）に対する Do の防御的ガードを検証します。
//
// 型ではゼロ値 Method{} を排除できないため、Do 実行時のガードで担保します。
func TestClientDo_EmptyMethod_Guard(t *testing.T) {
	t.Parallel()

	// ガードは Do の最初の文で、ネットワーク・registry 等に触れる前に返るため、ゼロ値 client で足りる。
	c := &client{}
	req := &Request{
		downstream: "empty-method",
		method:     Method{}, // ゼロ値（MethodGet() 等の定義済み値ではない）
		url:        "http://example.com",
	}

	resp, err := c.Do(context.Background(), req)

	require.ErrorIs(t, err, apperror.ErrInvalidArgument)
	assert.Nil(t, resp)
}
