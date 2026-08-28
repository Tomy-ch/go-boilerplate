package main

import (
	"context"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// queryInput は、検査が投げる形の Query 入力を返します。
func queryInput(kce string) *dynamodb.QueryInput {
	return &dynamodb.QueryInput{
		KeyConditionExpression: aws.String(kce),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s": &types.AttributeValueMemberS{Value: streamID},
			":c": &types.AttributeValueMemberN{Value: "2"},
		},
		ConsistentRead: aws.Bool(true),
	}
}

// newDDB は、fake を向いた DynamoDB 検査状態を返します。
func newDDB(t *testing.T) (*ddbSmoke, *awsFake) {
	t.Helper()

	f := newAWSFake(t)
	installDynamoDB(f)

	return &ddbSmoke{c: f.clients(t).dynamoDB, table: "gobp_smoke_test"}, f
}

func Test_runDynamoDB(t *testing.T) {
	t.Parallel()

	t.Run("全検査が互換なら D1〜D8 の 8 行", func(t *testing.T) {
		t.Parallel()

		s, _ := newDDB(t)
		rec := &recorder{}
		runDynamoDB(t.Context(), s.c, s.table, false, rec)

		require.Len(t, rec.results, 8)
		for _, r := range rec.results {
			assert.Equal(t, VerdictCompatible, r.Verdict, r.ID+" "+r.Detail)
		}

		assert.Equal(t, "D8", rec.results[7].ID)
	})

	t.Run("CreateTable が失敗すると後続は検証不能、後片付けも走らない", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("CreateTable", func(fakeRequest) fakeResponse { return jsonErr("InternalFailure", "boom") })

		rec := &recorder{}
		runDynamoDB(t.Context(), s.c, s.table, false, rec)

		require.Len(t, rec.results, 8)
		assert.Equal(t, VerdictIncompatible, rec.results[0].Verdict)
		assert.Equal(t, VerdictUnverifiable, rec.results[1].Verdict)
		assert.Contains(t, rec.results[7].Detail, "先行検査 D1")
		assert.NotContains(t, f.called(), "DeleteTable")
	})

	t.Run("後片付けは本体の ctx が cancel されていても走る", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		ctx, cancel := context.WithCancel(t.Context())
		f.on("UpdateTimeToLive", func(fakeRequest) fakeResponse {
			cancel() // 検査の途中で実行全体の期限が切れた状況を再現する

			return jsonOK(map[string]any{})
		})

		rec := &recorder{}
		runDynamoDB(ctx, s.c, s.table, false, rec)

		assert.Equal(t, VerdictCompatible, rec.results[7].Verdict, "D8 は独立の ctx で成功する")
		assert.Contains(t, f.called(), "DeleteTable")
	})
}

func Test_ddbSmoke_steps(t *testing.T) {
	t.Parallel()

	steps := (&ddbSmoke{}).steps()

	ids := make([]string, 0, len(steps))
	for _, st := range steps {
		ids = append(ids, st.id)
	}

	assert.Equal(t, []string{"D1", "D2", "D3", "D4", "D5", "D6", "D7"}, ids)
	assert.True(t, steps[0].halt, "table が無ければ以降は実行不能")
	assert.True(t, steps[1].halt, "最初の append が通らなければ順序検査は成立しない")
	assert.False(t, steps[2].halt)
}

func Test_ddbSmoke_createTable(t *testing.T) {
	t.Parallel()

	t.Run("作成して ACTIVE を待つ", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		detail, err := s.createTable(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "table gobp_smoke_test", detail)
		assert.Contains(t, f.called(), "DescribeTable")
	})

	t.Run("CreateTable の拒否はそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("CreateTable", func(fakeRequest) fakeResponse { return jsonErr("ResourceInUseException", "exists") })

		_, err := s.createTable(t.Context())

		var api apiError
		require.ErrorAs(t, err, &api)
		assert.Equal(t, "ResourceInUseException", api.ErrorCode())
	})
}

func Test_ddbSmoke_putSequence(t *testing.T) {
	t.Parallel()

	s, f := newDDB(t)

	var got fakeRequest
	var mu sync.Mutex
	st := installDynamoDB(f)
	f.on("PutItem", func(req fakeRequest) fakeResponse {
		mu.Lock()
		got = req
		mu.Unlock()

		return st.putItem(req)
	})

	require.NoError(t, s.putSequence(t.Context(), 7))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "7", jsonPath(got.json, "Item", attrSequence, "N"))
	assert.Equal(t, streamID, jsonPath(got.json, "Item", attrStream, "S"))
	assert.Equal(t, "attribute_not_exists("+attrStream+")", got.json["ConditionExpression"])
}

func Test_ddbSmoke_conditionalPut(t *testing.T) {
	t.Parallel()

	s, _ := newDDB(t)
	detail, err := s.conditionalPut(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "sequence 1 を append", detail)
}

func Test_ddbSmoke_duplicatePut(t *testing.T) {
	t.Parallel()

	t.Run("2 回目が型付きの条件エラーなら互換", func(t *testing.T) {
		t.Parallel()

		s, _ := newDDB(t)
		require.NoError(t, s.putSequence(t.Context(), 1))

		detail, err := s.duplicatePut(t.Context())
		require.NoError(t, err)
		assert.Contains(t, detail, codeConditionalCheckFailed)
	})

	t.Run("2 回目が成功してしまえば非互換", func(t *testing.T) {
		t.Parallel()

		s, _ := newDDB(t)

		_, err := s.duplicatePut(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "条件式が無視され")
	})

	t.Run("別のエラーはそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("PutItem", func(fakeRequest) fakeResponse { return jsonErr("ProvisionedThroughputExceededException", "slow") })

		_, err := s.duplicatePut(t.Context())

		var api apiError
		require.ErrorAs(t, err, &api)
		assert.Equal(t, "ProvisionedThroughputExceededException", api.ErrorCode())
	})
}

func Test_ddbSmoke_seed(t *testing.T) {
	t.Parallel()

	t.Run("2 から seedCount まで append する", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		require.NoError(t, s.seed(t.Context()))

		puts := 0
		for _, op := range f.called() {
			if op == "PutItem" {
				puts++
			}
		}

		assert.Equal(t, seedCount-1, puts)
	})

	t.Run("途中の失敗は sequence を添えて返す", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("PutItem", func(fakeRequest) fakeResponse { return jsonErr("InternalFailure", "boom") })

		err := s.seed(t.Context())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "seed sequence 2")
	})
}

func Test_ddbSmoke_query(t *testing.T) {
	t.Parallel()

	t.Run("#seq を使う式にだけ ExpressionAttributeNames を付ける", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)

		var mu sync.Mutex
		var seen []map[string]any
		f.on("Query", func(req fakeRequest) fakeResponse {
			mu.Lock()
			seen = append(seen, req.json)
			mu.Unlock()

			return fakeQuery(req)
		})

		_, _, err := s.query(t.Context(), queryInput("stream_id = :s AND #seq > :c"))
		require.NoError(t, err)
		_, _, err = s.query(t.Context(), queryInput("stream_id = :s"))
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, seen, 2)
		assert.NotNil(t, seen[0]["ExpressionAttributeNames"])
		assert.Nil(t, seen[1]["ExpressionAttributeNames"], "使わない名前を渡すと本物の DynamoDB は ValidationException を返す")
	})

	t.Run("sequence が N 型でなければ非互換", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("Query", func(fakeRequest) fakeResponse {
			return jsonOK(map[string]any{"Items": []map[string]any{{attrSequence: map[string]any{"S": "1"}}}})
		})

		_, _, err := s.query(t.Context(), queryInput("stream_id = :s"))
		assert.Equal(t, VerdictIncompatible, classify(err))
	})

	t.Run("sequence が整数でなければ非互換", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("Query", func(fakeRequest) fakeResponse {
			return jsonOK(map[string]any{"Items": []map[string]any{{attrSequence: map[string]any{"N": "1.5"}}}})
		})

		_, _, err := s.query(t.Context(), queryInput("stream_id = :s"))
		assert.Equal(t, VerdictIncompatible, classify(err))
	})

	t.Run("LastEvaluatedKey を返す", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("Query", func(fakeRequest) fakeResponse { return ddbPage([]int{1, 2}, 2) })

		seqs, last, err := s.query(t.Context(), queryInput("stream_id = :s"))
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, seqs)
		assert.NotNil(t, last)
	})
}

func Test_ddbSmoke_queryAfterCursor(t *testing.T) {
	t.Parallel()

	t.Run("cursor より後が昇順で返れば互換", func(t *testing.T) {
		t.Parallel()

		s, _ := newDDB(t)
		detail, err := s.queryAfterCursor(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "[3 4 5] が昇順で返った", detail)
	})

	t.Run("順序や欠番があれば非互換", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("Query", func(fakeRequest) fakeResponse { return ddbPage([]int{5, 4, 3}, 0) })

		_, err := s.queryAfterCursor(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "[5 4 3]")
	})
}

func Test_ddbSmoke_queryLatest(t *testing.T) {
	t.Parallel()

	t.Run("最新 1 件が返れば互換", func(t *testing.T) {
		t.Parallel()

		s, _ := newDDB(t)
		detail, err := s.queryLatest(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "sequence 5 が返った", detail)
	})

	t.Run("先頭が返れば非互換", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("Query", func(fakeRequest) fakeResponse { return ddbPage([]int{1}, 0) })

		_, err := s.queryLatest(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_ddbSmoke_queryPaginated(t *testing.T) {
	t.Parallel()

	t.Run("LastEvaluatedKey で継続して全件そろえば互換", func(t *testing.T) {
		t.Parallel()

		s, _ := newDDB(t)
		detail, err := s.queryPaginated(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "3 ページで 5 件（LastEvaluatedKey で継続）", detail)
	})

	t.Run("継続キーが返らず打ち切られれば非互換", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("Query", func(fakeRequest) fakeResponse { return ddbPage([]int{1, 2}, 0) })

		_, err := s.queryPaginated(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "1 ページ")
	})

	t.Run("継続キーが尽きなければ上限で止めて非互換", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		f.on("Query", func(fakeRequest) fakeResponse { return ddbPage([]int{1}, 1) })

		_, err := s.queryPaginated(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_ddbSmoke_timeToLive(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		attr    string
		status  string
		verdict Verdict
	}{
		"ENABLED が読み戻れば互換":  {attr: attrExpires, status: "ENABLED", verdict: VerdictCompatible},
		"ENABLING でも互換":     {attr: attrExpires, status: "ENABLING", verdict: VerdictCompatible},
		"DISABLED のままなら非互換": {attr: attrExpires, status: "DISABLED", verdict: VerdictIncompatible},
		"属性名が反映されなければ非互換":   {attr: "other", status: "ENABLED", verdict: VerdictIncompatible},
		"不明な状態は非互換":         {attr: attrExpires, status: "WEIRD", verdict: VerdictIncompatible},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s, f := newDDB(t)
			f.on("DescribeTimeToLive", ttlDescription(tt.attr, tt.status))

			detail, err := s.timeToLive(t.Context())
			assert.Equal(t, tt.verdict, classify(err))
			if tt.verdict == VerdictCompatible {
				assert.Contains(t, detail, tt.status)
			}
		})
	}
}

func Test_ddbSmoke_cleanup(t *testing.T) {
	t.Parallel()

	t.Run("table が作られていなければ検証不能として残す", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		rec := &recorder{}
		s.cleanup(t.Context(), false, false, rec)

		require.Len(t, rec.results, 1)
		assert.Equal(t, VerdictUnverifiable, rec.results[0].Verdict)
		assert.NotContains(t, f.called(), "DeleteTable")
	})

	t.Run("-keep なら削除せず table 名を残す", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		rec := &recorder{}
		s.cleanup(t.Context(), true, true, rec)

		assert.Equal(t, VerdictUnverifiable, rec.results[0].Verdict)
		assert.Contains(t, rec.results[0].Detail, s.table)
		assert.NotContains(t, f.called(), "DeleteTable")
	})

	t.Run("削除できれば互換", func(t *testing.T) {
		t.Parallel()

		s, f := newDDB(t)
		rec := &recorder{}
		s.cleanup(t.Context(), true, false, rec)

		assert.Equal(t, VerdictCompatible, rec.results[0].Verdict)
		assert.Contains(t, f.called(), "DeleteTable")
	})
}

func Test_equalInts(t *testing.T) {
	t.Parallel()

	assert.True(t, equalInts([]int{1, 2}, []int{1, 2}))
	assert.False(t, equalInts([]int{1, 2}, []int{2, 1}))
	assert.False(t, equalInts([]int{1}, []int{1, 2}))
	assert.True(t, equalInts(nil, []int{}))
}

func Test_joinInts(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "[3 4 5]", joinInts([]int{3, 4, 5}))
	assert.Equal(t, "[]", joinInts(nil))
	assert.NotContains(t, joinInts([]int{1}), ",")
}
