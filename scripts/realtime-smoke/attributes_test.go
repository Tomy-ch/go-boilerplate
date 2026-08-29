package main

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pubSubSmoke_attributeSteps(t *testing.T) {
	t.Parallel()

	steps := (&pubSubSmoke{}).attributeSteps()

	ids := make([]string, 0, len(steps))
	for _, st := range steps {
		ids = append(ids, st.id)
		assert.False(t, st.halt, st.id+" は fan-out の成否と独立なので後続を止めない")
	}

	assert.Equal(t, []string{"G10", "G11", "G12", "G13", "G14", "G15", "G16", "G17"}, ids)
}

func Test_pubSubSmoke_setAndReadBack(t *testing.T) {
	t.Parallel()

	attrs := map[sqstypes.QueueAttributeName]string{sqstypes.QueueAttributeNameVisibilityTimeout: "30"}

	t.Run("同じ値が読み戻れば nil", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 1)
		require.NoError(t, s.setAndReadBack(t.Context(), s.queueURLs[0], attrs))
	})

	t.Run("JSON 属性は整形差を無視して比較する", func(t *testing.T) {
		t.Parallel()

		s, f, st := prepared(t, 1)
		f.on("SetQueueAttributes", func(fakeRequest) fakeResponse {
			st.attrs[s.queueURLs[0]] = map[string]string{"RedrivePolicy": `{"maxReceiveCount": "5", "deadLetterTargetArn": "a"}`}

			return jsonOK(map[string]any{})
		})

		require.NoError(t, s.setAndReadBack(t.Context(), s.queueURLs[0], map[sqstypes.QueueAttributeName]string{
			sqstypes.QueueAttributeNameRedrivePolicy: `{"deadLetterTargetArn":"a","maxReceiveCount":"5"}`,
		}))
	})

	t.Run("読み戻せなければ silent drop として非互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("SetQueueAttributes", func(fakeRequest) fakeResponse { return jsonOK(map[string]any{}) })

		err := s.setAndReadBack(t.Context(), s.queueURLs[0], attrs)
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "silent drop")
	})

	t.Run("値が違えば非互換", func(t *testing.T) {
		t.Parallel()

		s, f, st := prepared(t, 1)
		f.on("SetQueueAttributes", func(fakeRequest) fakeResponse {
			st.attrs[s.queueURLs[0]] = map[string]string{"VisibilityTimeout": "31"}

			return jsonOK(map[string]any{})
		})

		err := s.setAndReadBack(t.Context(), s.queueURLs[0], attrs)
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "異なる")
	})

	t.Run("拒否されれば API エラーをそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("SetQueueAttributes", func(fakeRequest) fakeResponse { return jsonErr(fakeErrPolicy, "invalid") })

		err := s.setAndReadBack(t.Context(), s.queueURLs[0], attrs)

		var api apiError
		require.ErrorAs(t, err, &api)
		assert.Equal(t, fakeErrPolicy, api.ErrorCode())
	})
}

func Test_pubSubSmoke_firstQueue(t *testing.T) {
	t.Parallel()

	t.Run("先頭の queue URL を返す", func(t *testing.T) {
		t.Parallel()

		url, err := (&pubSubSmoke{queueURLs: []string{"a", "b"}}).firstQueue()
		require.NoError(t, err)
		assert.Equal(t, "a", url)
	})

	t.Run("queue が無ければ非互換", func(t *testing.T) {
		t.Parallel()

		_, err := (&pubSubSmoke{}).firstQueue()
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_queueTimings(t *testing.T) {
	t.Parallel()

	t.Run("2 属性が読み戻れば互換", func(t *testing.T) {
		t.Parallel()

		s, _, st := prepared(t, 1)
		detail, err := s.queueTimings(t.Context())

		require.NoError(t, err)
		assert.Contains(t, detail, "VisibilityTimeout=30")
		assert.Equal(t, "20", st.attrs[s.queueURLs[0]]["ReceiveMessageWaitTimeSeconds"])
	})

	t.Run("queue が無ければ非互換", func(t *testing.T) {
		t.Parallel()

		_, err := (&pubSubSmoke{}).queueTimings(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_redrive(t *testing.T) {
	t.Parallel()

	t.Run("DLQ を作り、その ARN を含む RedrivePolicy が読み戻れば互換", func(t *testing.T) {
		t.Parallel()

		s, _, st := prepared(t, 1)
		detail, err := s.redrive(t.Context())

		require.NoError(t, err)
		assert.Contains(t, detail, "maxReceiveCount=5")
		assert.Equal(t, st.queueURL(s.names.dlq()), s.dlqURL, "後片付けのために DLQ の URL を保持する")
		assert.Contains(t, st.attrs[s.queueURLs[0]]["RedrivePolicy"], fakeQueue+s.names.dlq())
	})

	t.Run("DLQ の作成が拒否されればそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("CreateQueue", func(fakeRequest) fakeResponse { return jsonErr("QueueNameExists", "dup") })

		_, err := s.redrive(t.Context())

		var api apiError
		require.ErrorAs(t, err, &api)
		assert.Empty(t, s.dlqURL)
	})

	t.Run("queue が無ければ非互換", func(t *testing.T) {
		t.Parallel()

		_, err := (&pubSubSmoke{}).redrive(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_managedSSE(t *testing.T) {
	t.Parallel()

	t.Run("true が読み戻れば互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 1)
		detail, err := s.managedSSE(t.Context())

		require.NoError(t, err)
		assert.Contains(t, detail, "SqsManagedSseEnabled=true")
	})

	t.Run("queue が無ければ非互換", func(t *testing.T) {
		t.Parallel()

		_, err := (&pubSubSmoke{}).managedSSE(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_kmsKey(t *testing.T) {
	t.Parallel()

	t.Run("alias が読み戻れば互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 1)
		detail, err := s.kmsKey(t.Context())

		require.NoError(t, err)
		assert.Contains(t, detail, kmsKeyAlias)
	})

	t.Run("queue が無ければ非互換", func(t *testing.T) {
		t.Parallel()

		_, err := (&pubSubSmoke{}).kmsKey(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_listSubscriptions(t *testing.T) {
	t.Parallel()

	t.Run("queue ARN から Subscribe と同じ ARN が引ければ互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 2)
		detail, err := s.listSubscriptions(t.Context())

		require.NoError(t, err)
		assert.Contains(t, detail, s.subArns[0])
	})

	t.Run("一覧に無ければ非互換", func(t *testing.T) {
		t.Parallel()

		s, _, st := prepared(t, 1)
		for arn := range st.subs {
			delete(st.subs, arn)
		}

		_, err := s.listSubscriptions(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "一覧に無い")
	})

	t.Run("ARN が Subscribe の戻り値と違えば非互換", func(t *testing.T) {
		t.Parallel()

		s, _, _ := prepared(t, 1)
		s.subArns[0] = "arn:other"

		_, err := s.listSubscriptions(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "異なる")
	})

	t.Run("先行検査を通っていなければ非互換", func(t *testing.T) {
		t.Parallel()

		_, err := (&pubSubSmoke{}).listSubscriptions(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})

	t.Run("一覧の取得失敗はそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("ListSubscriptionsByTopic", func(fakeRequest) fakeResponse { return xmlErr("NotFound", "topic") })

		_, err := s.listSubscriptions(t.Context())

		var api apiError
		require.ErrorAs(t, err, &api)
	})
}

func Test_pubSubSmoke_notificationTypes(t *testing.T) {
	t.Parallel()

	t.Run("wakeup と revocation が 1 件ずつ属性で振り分けられ、削除まで行えば互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		detail, err := s.notificationTypes(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "2 件を type 属性で振り分けた（wakeup 1 / revocation 1）", detail)
		assert.Contains(t, f.called(), "DeleteMessage")
	})

	t.Run("属性が落ちれば非互換", func(t *testing.T) {
		t.Parallel()

		s, f, st := prepared(t, 1)
		publish := st.publish
		f.on("Publish", func(req fakeRequest) fakeResponse {
			req.form.Del("MessageAttributes.entry.1.Name")

			return publish(req)
		})

		_, err := s.notificationTypes(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "type 属性の無い")
	})

	t.Run("Publish の失敗はそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("Publish", func(fakeRequest) fakeResponse { return xmlErr("NotFound", "topic") })

		_, err := s.notificationTypes(t.Context())

		var api apiError
		require.ErrorAs(t, err, &api)
	})

	t.Run("queue が無ければ非互換", func(t *testing.T) {
		t.Parallel()

		_, err := (&pubSubSmoke{}).notificationTypes(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_pubSubSmoke_receiveAfterAttributes(t *testing.T) {
	t.Parallel()

	t.Run("1 件だけ届き削除まで行えば互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		detail, err := s.receiveAfterAttributes(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "属性設定後も 1 件だけ届いた", detail)
		assert.Contains(t, f.called(), "DeleteMessage")
	})

	t.Run("1 件も届かなければ配送停止として非互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("Publish", func(fakeRequest) fakeResponse { return xmlOK("Publish", "<MessageId>m-1</MessageId>") })

		_, err := s.receiveAfterAttributes(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "配送停止")
	})

	t.Run("同じ message が繰り返し届けば非互換", func(t *testing.T) {
		t.Parallel()

		s, f, st := prepared(t, 1)
		publish := st.publish
		f.on("Publish", func(req fakeRequest) fakeResponse {
			publish(req)

			return publish(req)
		})

		_, err := s.receiveAfterAttributes(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "重複配送")
	})

	t.Run("受信した message の削除が拒否されれば非互換", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("DeleteMessage", func(fakeRequest) fakeResponse { return jsonErr("QueueExists", "not in queue") })

		_, err := s.receiveAfterAttributes(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
		assert.Contains(t, err.Error(), "削除できない")
	})

	t.Run("Publish の失敗はそのまま返す", func(t *testing.T) {
		t.Parallel()

		s, f, _ := prepared(t, 1)
		f.on("Publish", func(fakeRequest) fakeResponse { return xmlErr("NotFound", "topic") })

		_, err := s.receiveAfterAttributes(t.Context())

		var api apiError
		require.ErrorAs(t, err, &api)
	})

	t.Run("queue が無ければ非互換", func(t *testing.T) {
		t.Parallel()

		_, err := (&pubSubSmoke{}).receiveAfterAttributes(t.Context())
		assert.Equal(t, VerdictIncompatible, classify(err))
	})
}

func Test_dispatchByType(t *testing.T) {
	t.Parallel()

	msg := func(kind, body string) sqstypes.Message {
		m := sqstypes.Message{Body: aws.String(body)}
		if kind != "" {
			m.MessageAttributes = map[string]sqstypes.MessageAttributeValue{attrNotificationType: {StringValue: aws.String(kind)}}
		}

		return m
	}
	wakeup := `{"eventId":"e","streamId":"s","sequence":"2"}`
	revoked := `{"subject":"u","destination":"s"}`

	tests := map[string]struct {
		msgs     []sqstypes.Message
		verdict  Verdict
		contains string
	}{
		"1 件ずつ揃えば互換": {
			msgs:     []sqstypes.Message{msg(typeWakeup, wakeup), msg(typeRevocation, revoked)},
			verdict:  VerdictCompatible,
			contains: "wakeup 1 / revocation 1",
		},
		"属性が無ければ非互換":            {msgs: []sqstypes.Message{msg("", wakeup)}, verdict: VerdictIncompatible, contains: "type 属性の無い"},
		"未知の種別は非互換":             {msgs: []sqstypes.Message{msg("other", wakeup)}, verdict: VerdictIncompatible, contains: "未知の type"},
		"wakeup の body が違えば非互換": {msgs: []sqstypes.Message{msg(typeWakeup, revoked)}, verdict: VerdictIncompatible, contains: "通知の形でない"},
		"片方しか届かなければ非互換":         {msgs: []sqstypes.Message{msg(typeWakeup, wakeup)}, verdict: VerdictIncompatible, contains: "revocation 0 件"},
		"重複すれば非互換": {
			msgs:     []sqstypes.Message{msg(typeWakeup, wakeup), msg(typeWakeup, wakeup), msg(typeRevocation, revoked)},
			verdict:  VerdictIncompatible,
			contains: "wakeup 2 件",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			detail, err := dispatchByType(tt.msgs)

			assert.Equal(t, tt.verdict, classify(err))
			if err != nil {
				assert.Contains(t, err.Error(), tt.contains)
			} else {
				assert.Contains(t, detail, tt.contains)
			}
		})
	}
}

func Test_isNotification(t *testing.T) {
	t.Parallel()

	assert.True(t, isNotification(`{"eventId":"e","streamId":"s","sequence":"1"}`))
	assert.False(t, isNotification(`{"subject":"u","destination":"s"}`))
	assert.False(t, isNotification(`{"eventId":"e","streamId":"s"}`), "sequence が欠ける")
	assert.False(t, isNotification("not json"))
}
