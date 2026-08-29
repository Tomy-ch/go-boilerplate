package main

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRunID = "0123456789ab"

// newPubSub は、fake を向いた SNS / SQS 検査状態を返します。
func newPubSub(t *testing.T, subscribers int) (*pubSubSmoke, *awsFake, *goawsState) {
	t.Helper()

	f := newAWSFake(t)
	st := installGoAWS(f)
	c := f.clients(t)

	return &pubSubSmoke{sns: c.sns, sqs: c.sqs, names: names{runID: testRunID}, n: subscribers, window: time.Second}, f, st
}

// prepared は、G5 まで通した状態（topic / queue / subscription 作成済み）を返します。
func prepared(t *testing.T, subscribers int) (*pubSubSmoke, *awsFake, *goawsState) {
	t.Helper()

	s, f, st := newPubSub(t, subscribers)
	for _, fn := range []func(context.Context) (string, error){s.createTopic, s.createQueues, s.queueArnsStep, s.subscribe} {
		_, err := fn(t.Context())
		require.NoError(t, err)
	}

	return s, f, st
}

func Test_runPubSub(t *testing.T) {
	t.Parallel()

	t.Run("全検査が互換なら G0〜G17 の 19 行", func(t *testing.T) {
		t.Parallel()

		f := newAWSFake(t)
		installGoAWS(f)
		c := f.clients(t)

		rec := &recorder{}
		runPubSub(t.Context(), c.sns, c.sqs, names{runID: testRunID}, 2, false, rec)

		require.Len(t, rec.results, 19)
		for _, r := range rec.results {
			assert.Equal(t, VerdictCompatible, r.Verdict, r.ID+" "+r.Detail)
		}

		assert.Equal(t, "G9", rec.results[18].ID)
	})

	t.Run("Policy を拒否する emulator では G4 / G4b だけが非互換", func(t *testing.T) {
		t.Parallel()

		f := newAWSFake(t)
		installGoAWS(f).rejectPolicy = true
		c := f.clients(t)

		rec := &recorder{}
		runPubSub(t.Context(), c.sns, c.sqs, names{runID: testRunID}, 1, false, rec)

		for _, r := range rec.results {
			if r.ID == "G4" || r.ID == "G4b" {
				assert.Equal(t, VerdictIncompatible, r.Verdict, r.ID)
				assert.Contains(t, r.Detail, "InvalidParameterValue")
			} else {
				assert.Equal(t, VerdictCompatible, r.Verdict, r.ID+" "+r.Detail)
			}
		}
	})

	t.Run("protocol probe が未対応なら後続は検証不能、後片付けは対象なし", func(t *testing.T) {
		t.Parallel()

		f := newAWSFake(t)
		installGoAWS(f)
		f.on("ListQueues", func(fakeRequest) fakeResponse { return jsonErr("InvalidAction", "unknown") })
		c := f.clients(t)

		rec := &recorder{}
		runPubSub(t.Context(), c.sns, c.sqs, names{runID: testRunID}, 1, false, rec)

		require.Len(t, rec.results, 19)
		assert.Equal(t, VerdictUnsupported, rec.results[0].Verdict)
		assert.Equal(t, VerdictUnverifiable, rec.results[1].Verdict)
		assert.Contains(t, rec.results[18].Detail, "作成された resource が無い")
	})
}

func Test_pubSubSmoke_steps(t *testing.T) {
	t.Parallel()

	steps := (&pubSubSmoke{}).steps()

	ids := make([]string, 0, len(steps))
	halts := map[string]bool{}
	for _, st := range steps {
		ids = append(ids, st.id)
		halts[st.id] = st.halt
	}

	assert.Equal(
		t,
		[]string{"G0", "G1", "G2", "G3", "G4", "G4b", "G5", "G6", "G7", "G8", "G10", "G11", "G12", "G13", "G14", "G15", "G16", "G17"},
		ids,
	)
	assert.True(t, halts["G0"], "wire protocol が通らなければ全部が実行不能")
	assert.True(t, halts["G5"], "subscription 無しでは fan-out を検証できない")
	assert.False(t, halts["G4"], "policy の可否は fan-out の成否と独立")
}

func Test_pubSubSmoke_probeProtocol(t *testing.T) {
	t.Parallel()

	t.Run("JSON protocol で応答すれば互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := newPubSub(t, 1)
		detail, err := s.probeProtocol(t.Context())

		require.NoError(t, err)
		assert.Contains(t, detail, "JSON protocol")
	})

	t.Run("XML が返れば deserialization failed として未対応", func(t *testing.T) {
		t.Parallel()

		s, f, _ := newPubSub(t, 1)
		f.on("ListQueues", func(fakeRequest) fakeResponse { return xmlOK("ListQueues", "") })

		_, err := s.probeProtocol(t.Context())
		assert.Equal(t, VerdictUnsupported, classify(err))
	})
}

func Test_pubSubSmoke_createTopic(t *testing.T) {
	t.Parallel()

	t.Run("TopicArn を保持する", func(t *testing.T) {
		t.Parallel()

		s, _, _ := newPubSub(t, 1)
		detail, err := s.createTopic(t.Context())

		require.NoError(t, err)
		assert.Equal(t, fakeTopic+"gobp-smoke-"+testRunID, s.topicArn)
		assert.Equal(t, s.topicArn, detail)
	})

	t.Run("TopicArn が空なら非互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := newPubSub(t, 1)
		f.on("CreateTopic", func(fakeRequest) fakeResponse { return xmlOK("CreateTopic", "") })

		_, err := s.createTopic(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_createQueues(t *testing.T) {
	t.Parallel()

	t.Run("N 個作って URL を保持する", func(t *testing.T) {
		t.Parallel()

		s, _, _ := newPubSub(t, 3)
		detail, err := s.createQueues(t.Context())

		require.NoError(t, err)
		require.Len(t, s.queueURLs, 3)
		assert.Contains(t, detail, "3 queue")
		assert.Contains(t, s.queueURLs[2], "gobp-smoke-"+testRunID+"-2")
	})

	t.Run("QueueUrl が空なら非互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := newPubSub(t, 1)
		f.on("CreateQueue", func(fakeRequest) fakeResponse { return jsonOK(map[string]any{}) })

		_, err := s.createQueues(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_queueAttribute(t *testing.T) {
	t.Parallel()

	s, _, _ := newPubSub(t, 1)
	_, err := s.createQueues(t.Context())
	require.NoError(t, err)

	arn, err := s.queueAttribute(t.Context(), s.queueURLs[0], sqstypes.QueueAttributeNameQueueArn)
	require.NoError(t, err)
	assert.Equal(t, fakeQueue+"gobp-smoke-"+testRunID+"-0", arn)

	policy, err := s.queueAttribute(t.Context(), s.queueURLs[0], sqstypes.QueueAttributeNamePolicy)
	require.NoError(t, err)
	assert.Empty(t, policy, "未設定の属性は空")
}

func Test_pubSubSmoke_queueArnsStep(t *testing.T) {
	t.Parallel()

	t.Run("全 queue の ARN を保持する", func(t *testing.T) {
		t.Parallel()

		s, _, _ := newPubSub(t, 2)
		_, err := s.createQueues(t.Context())
		require.NoError(t, err)

		_, err = s.queueArnsStep(t.Context())
		require.NoError(t, err)
		assert.Len(t, s.queueArns, 2)
	})

	t.Run("ARN が空なら非互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := newPubSub(t, 1)
		_, err := s.createQueues(t.Context())
		require.NoError(t, err)
		f.on("GetQueueAttributes", func(fakeRequest) fakeResponse { return jsonOK(map[string]any{"Attributes": map[string]string{}}) })

		_, err = s.queueArnsStep(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_policyDocument(t *testing.T) {
	t.Parallel()

	s := &pubSubSmoke{topicArn: "arn:topic"}
	doc, err := s.policyDocument("arn:queue")
	require.NoError(t, err)

	var parsed queuePolicy
	require.NoError(t, json.Unmarshal([]byte(doc), &parsed))
	require.Len(t, parsed.Statement, 1)
	assert.Equal(t, "arn:queue", parsed.Statement[0].Resource)
	assert.Equal(t, "sqs:SendMessage", parsed.Statement[0].Action)
	assert.Contains(t, doc, `"aws:SourceArn":"arn:topic"`)
}

func Test_pubSubSmoke_verifyPolicy(t *testing.T) {
	t.Parallel()

	t.Run("同じ値が読み戻れば nil", func(t *testing.T) {
		t.Parallel()

		s, _, st := newPubSub(t, 1)
		_, err := s.createQueues(t.Context())
		require.NoError(t, err)
		st.policies[s.queueURLs[0]] = `{"Version": "2012-10-17", "Statement": []}`

		require.NoError(t, s.verifyPolicy(t.Context(), s.queueURLs[0], `{"Statement":[],"Version":"2012-10-17"}`))
	})

	t.Run("読み戻せなければ silent drop として非互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := newPubSub(t, 1)
		_, err := s.createQueues(t.Context())
		require.NoError(t, err)

		err = s.verifyPolicy(t.Context(), s.queueURLs[0], `{}`)
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "silent drop")
	})

	t.Run("値が違えば非互換", func(t *testing.T) {
		t.Parallel()

		s, _, st := newPubSub(t, 1)
		_, err := s.createQueues(t.Context())
		require.NoError(t, err)
		st.policies[s.queueURLs[0]] = `{"Version":"other"}`

		err = s.verifyPolicy(t.Context(), s.queueURLs[0], `{"Version":"2012-10-17"}`)
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_queuePolicy(t *testing.T) {
	t.Parallel()

	t.Run("保存され読み戻せれば互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 2)
		detail, err := s.queuePolicy(t.Context())

		require.NoError(t, err)
		assert.Contains(t, detail, "aws:SourceArn")
	})

	t.Run("拒否されれば API エラーをそのまま返す（非互換）", func(t *testing.T) {
		t.Parallel()

		s, _, st := prepared(t, 1)
		st.rejectPolicy = true

		_, err := s.queuePolicy(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, describe(err), "InvalidParameterValue")
	})
}

func Test_pubSubSmoke_createQueueWithPolicy(t *testing.T) {
	t.Parallel()

	t.Run("作成時の Policy が読み戻せれば互換で、queue は削除される", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		detail, err := s.createQueueWithPolicy(t.Context())

		require.NoError(t, err)
		assert.Contains(t, detail, "作成時の Policy")
		assert.Contains(t, f.called(), "DeleteQueue")
	})

	t.Run("拒否されれば非互換", func(t *testing.T) {
		t.Parallel()

		s, _, st := prepared(t, 1)
		st.rejectPolicy = true

		_, err := s.createQueueWithPolicy(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})

	t.Run("読み戻しに失敗しても queue は削除してから非互換を返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("GetQueueAttributes", func(fakeRequest) fakeResponse { return jsonOK(map[string]any{"Attributes": map[string]string{}}) })

		_, err := s.createQueueWithPolicy(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, f.called(), "DeleteQueue")
	})
}

func Test_pubSubSmoke_subscribe(t *testing.T) {
	t.Parallel()

	t.Run("RawMessageDelivery=true が読み戻れば互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := newPubSub(t, 2)
		for _, fn := range []func(context.Context) (string, error){s.createTopic, s.createQueues, s.queueArnsStep} {
			_, err := fn(t.Context())
			require.NoError(t, err)
		}

		detail, err := s.subscribe(t.Context())
		require.NoError(t, err)
		assert.Len(t, s.subArns, 2)
		assert.Contains(t, detail, "2 subscription")
	})

	t.Run("属性が反映されなければ非互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := newPubSub(t, 1)
		for _, fn := range []func(context.Context) (string, error){s.createTopic, s.createQueues, s.queueArnsStep} {
			_, err := fn(t.Context())
			require.NoError(t, err)
		}

		f.on("SetSubscriptionAttributes", func(fakeRequest) fakeResponse { return xmlOK("SetSubscriptionAttributes", "") })

		_, err := s.subscribe(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), attrRawMessageDelivery)
	})
}

func Test_pubSubSmoke_publishAndReceive(t *testing.T) {
	t.Parallel()

	t.Run("全 queue に raw payload が届けば互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 2)
		detail, err := s.publishAndReceive(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "2 queue すべてに raw payload が 1 件ずつ届いた", detail)
		assert.JSONEq(t, `{"eventId":"evt-`+testRunID+`","streamId":"smoke","sequence":"1"}`, s.payload)
	})

	t.Run("envelope 付きで届けば非互換", func(t *testing.T) {
		t.Parallel()

		s, _, st := prepared(t, 1)
		for arn := range st.raw {
			st.raw[arn] = false
		}

		_, err := s.publishAndReceive(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "envelope")
	})

	t.Run("Publish の失敗はそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("Publish", func(fakeRequest) fakeResponse { return xmlErr("NotFound", "topic") })

		_, err := s.publishAndReceive(t.Context())

		var api apiError
		require.ErrorAs(t, err, &api)
		assert.Equal(t, "NotFound", api.ErrorCode())
	})
}

func Test_pubSubSmoke_receive(t *testing.T) {
	t.Parallel()

	t.Run("最初の batch の後にもう 1 回 poll して重複を拾う", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 1)

		var mu sync.Mutex
		polls := 0
		f2 := newAWSFake(t)
		f2.on("ReceiveMessage", func(fakeRequest) fakeResponse {
			mu.Lock()
			defer mu.Unlock()

			polls++
			if polls <= 2 {
				return jsonOK(map[string]any{"Messages": []map[string]any{{"MessageId": strconv.Itoa(polls), "Body": "x"}}})
			}

			return jsonOK(map[string]any{"Messages": []any{}})
		})
		s.sqs = f2.clients(t).sqs

		msgs, err := s.receive(t.Context(), s.queueURLs[0])
		require.NoError(t, err)
		assert.Len(t, msgs, 2, "1 回目の 1 件 + drain の 1 件")

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 2, polls)
	})

	t.Run("窓の中に届かなければ空を返す（非互換の判定は呼び出し元）", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 1)
		s.window = 50 * time.Millisecond

		msgs, err := s.receive(t.Context(), s.queueURLs[0])
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("実行全体の期限切れは transport 失敗として返す", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 1)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		_, err := s.receive(ctx, s.queueURLs[0])
		require.Error(t, err)
		assert.Equal(t, VerdictUnverifiable, classify(err))
	})

	t.Run("サーバーの拒否はそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("ReceiveMessage", func(fakeRequest) fakeResponse { return jsonErr("QueueDoesNotExist", "gone") })

		_, err := s.receive(t.Context(), s.queueURLs[0])

		var api apiError
		require.ErrorAs(t, err, &api)
	})
}

func Test_pubSubSmoke_verifyDelivery(t *testing.T) {
	t.Parallel()

	payload := `{"eventId":"e","streamId":"smoke","sequence":"1"}`
	msg := func(body string) sqstypes.Message { return sqstypes.Message{Body: aws.String(body)} }

	tests := map[string]struct {
		received [][]sqstypes.Message
		verdict  Verdict
		contains string
	}{
		"全 queue に 1 件ずつ": {received: [][]sqstypes.Message{{msg(payload)}, {msg(payload)}}, verdict: VerdictCompatible, contains: "2 queue"},
		"届かない queue がある":  {received: [][]sqstypes.Message{{msg(payload)}, {}}, verdict: VerdictIncompatible, contains: "1 queue にしか"},
		"重複配送":            {received: [][]sqstypes.Message{{msg(payload), msg(payload)}}, verdict: VerdictIncompatible, contains: "重複"},
		"SNS envelope が届く": {
			received: [][]sqstypes.Message{{msg(`{"Type":"Notification","Message":"x"}`)}},
			verdict:  VerdictIncompatible,
			contains: "envelope",
		},
		"body が違う": {received: [][]sqstypes.Message{{msg("other")}}, verdict: VerdictIncompatible, contains: "一致しない"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := &pubSubSmoke{payload: payload, received: tt.received}
			detail, err := s.verifyDelivery()

			assert.Equal(t, tt.verdict, classify(err))
			if err != nil {
				assert.Contains(t, err.Error(), tt.contains)
			} else {
				assert.Contains(t, detail, tt.contains)
			}
		})
	}
}

func Test_isSNSEnvelope(t *testing.T) {
	t.Parallel()

	assert.True(t, isSNSEnvelope(`{"Type":"Notification","Message":"x"}`))
	assert.False(t, isSNSEnvelope(`{"eventId":"e"}`))
	assert.False(t, isSNSEnvelope("not json"))
}

func Test_pubSubSmoke_messageAttributes(t *testing.T) {
	t.Parallel()

	attr := func(v string) map[string]sqstypes.MessageAttributeValue {
		return map[string]sqstypes.MessageAttributeValue{attrEventType: {StringValue: aws.String(v)}}
	}

	tests := map[string]struct {
		received [][]sqstypes.Message
		verdict  Verdict
	}{
		"属性が透過すれば互換":           {received: [][]sqstypes.Message{{{MessageAttributes: attr(eventTypeValue)}}}, verdict: VerdictCompatible},
		"message を受けていなければ非互換": {received: nil, verdict: VerdictIncompatible},
		"空の queue があれば非互換":     {received: [][]sqstypes.Message{{}}, verdict: VerdictIncompatible},
		"属性が落ちれば非互換":           {received: [][]sqstypes.Message{{{}}}, verdict: VerdictIncompatible},
		"値が変われば非互換":            {received: [][]sqstypes.Message{{{MessageAttributes: attr("other")}}}, verdict: VerdictIncompatible},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := &pubSubSmoke{received: tt.received}
			_, err := s.messageAttributes(t.Context())
			assert.Equal(t, tt.verdict, classify(err))
		})
	}
}

func Test_pubSubSmoke_deleteMessages(t *testing.T) {
	t.Parallel()

	t.Run("受けた message を全て削除する", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 2)
		_, err := s.publishAndReceive(t.Context())
		require.NoError(t, err)

		detail, err := s.deleteMessages(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "2 件を削除", detail)
		assert.Contains(t, f.called(), "DeleteMessage")
	})

	t.Run("message を受けていなければ非互換", func(t *testing.T) {
		t.Parallel()

		s := &pubSubSmoke{}
		_, err := s.deleteMessages(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_cleanup(t *testing.T) {
	t.Parallel()

	t.Run("何も作られていなければ検証不能として残す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := newPubSub(t, 1)
		rec := &recorder{}
		s.cleanup(t.Context(), false, rec)

		assert.Equal(t, VerdictUnverifiable, rec.results[0].Verdict)
		assert.NotContains(t, f.called(), "DeleteTopic")
	})

	t.Run("-keep なら削除せず topic を残す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		rec := &recorder{}
		s.cleanup(t.Context(), true, rec)

		assert.Equal(t, VerdictUnverifiable, rec.results[0].Verdict)
		assert.Contains(t, rec.results[0].Detail, s.topicArn)
		assert.NotContains(t, f.called(), "DeleteTopic")
	})

	t.Run("削除できれば互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		rec := &recorder{}
		s.cleanup(t.Context(), false, rec)

		assert.Equal(t, VerdictCompatible, rec.results[0].Verdict)
		assert.Contains(t, f.called(), "DeleteTopic")
	})
}

func Test_pubSubSmoke_teardown(t *testing.T) {
	t.Parallel()

	t.Run("unsubscribe → queue → topic の順で全部消す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 2)
		require.NoError(t, s.teardown(t.Context()))

		calls := f.called()
		assert.Equal(t, []string{"Unsubscribe", "Unsubscribe", "DeleteQueue", "DeleteQueue", "DeleteTopic"}, calls[len(calls)-5:])
	})

	t.Run("DLQ を作っていればそれも削除する", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		_, err := s.redrive(t.Context())
		require.NoError(t, err)

		require.NoError(t, s.teardown(t.Context()))

		calls := f.called()
		assert.Equal(t, []string{"Unsubscribe", "DeleteQueue", "DeleteQueue", "DeleteTopic"}, calls[len(calls)-4:])
	})

	t.Run("途中で失敗しても残りを試み、失敗をまとめて返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 2)
		f.on("Unsubscribe", func(fakeRequest) fakeResponse { return xmlErr("InternalError", "boom") })

		err := s.teardown(t.Context())
		require.Error(t, err)
		assert.Contains(t, f.called(), "DeleteQueue")
		assert.Contains(t, f.called(), "DeleteTopic")
		assert.Equal(t, 2, countOf(f.called(), "Unsubscribe"))
	})
}

func countOf(ops []string, op string) int {
	n := 0
	for _, o := range ops {
		if o == op {
			n++
		}
	}

	return n
}

func Test_sameJSON(t *testing.T) {
	t.Parallel()

	assert.True(t, sameJSON(`{"a":1,"b":[1,2]}`, `{ "b": [1, 2], "a": 1 }`))
	assert.False(t, sameJSON(`{"a":1}`, `{"a":2}`))
	assert.False(t, sameJSON(`{"a":1}`, `not json`))
	assert.False(t, sameJSON(`not json`, `{"a":1}`))
}

func Test_canonicalJSON(t *testing.T) {
	t.Parallel()

	got, err := canonicalJSON(`{ "b": 2, "a": {"y": 1, "x": 0} }`)
	require.NoError(t, err)
	assert.Equal(t, `{"a":{"x":0,"y":1},"b":2}`, got) //nolint:testifylint // key 順の正規化そのものを検証する

	_, err = canonicalJSON(`{`)
	require.Error(t, err)
}

func Test_pubSubSmoke_pollOnce(t *testing.T) {
	t.Parallel()

	t.Run("届いた message をそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, _, st := prepared(t, 1)
		st.inbox[s.queueURLs[0]] = []map[string]any{{"MessageId": "a", "Body": "1"}}

		msgs, err := s.pollOnce(t.Context(), s.queueURLs[0], receiveWait)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "1", aws.ToString(msgs[0].Body))
	})

	t.Run("サーバーの拒否はそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("ReceiveMessage", func(fakeRequest) fakeResponse { return jsonErr("QueueDoesNotExist", "gone") })

		_, err := s.pollOnce(t.Context(), s.queueURLs[0], drainWait)

		var api apiError
		require.ErrorAs(t, err, &api)
		assert.Equal(t, "QueueDoesNotExist", api.ErrorCode())
	})
}
