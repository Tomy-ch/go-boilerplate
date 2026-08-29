package main

// AWS の 3 サービスを 1 つの httptest.Server で模す。SDK の wire（DynamoDB / SQS は AWS JSON 1.0 と
// X-Amz-Target、SNS は Query + XML）に合わせて応答し、検査コードを emulator 無しで通す。

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	fakeAccount = "000000000000"
	fakeTopic   = "arn:aws:sns:us-east-1:" + fakeAccount + ":"
	fakeQueue   = "arn:aws:sqs:us-east-1:" + fakeAccount + ":"
	// fakeErrPolicy は、GoAWS が Policy 属性に返す実際のエラーコードです。
	fakeErrPolicy = "AWS.SimpleQueueService.InvalidParameterValue"
	// fakeErrCondition は、DynamoDB が条件式の不成立に返す __type です。
	fakeErrCondition = "com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException"
)

type fakeRequest struct {
	op   string
	json map[string]any
	form url.Values
}

type fakeResponse struct {
	status      int
	contentType string
	body        string
}

type fakeHandler func(req fakeRequest) fakeResponse

// awsFake は、操作名 → handler の表で応答する AWS もどきです。
type awsFake struct {
	t        *testing.T
	server   *httptest.Server
	mu       sync.Mutex
	handlers map[string]fakeHandler
	calls    []string
}

// ddbState は、DynamoDB fake の状態です。conditional put の重複を検出するため key を覚えます。
type ddbState struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

// goawsState は、GoAWS fake の状態です。rejectPolicy で実物と同じ Policy 拒否を再現します。
type goawsState struct {
	mu           sync.Mutex
	rejectPolicy bool
	policies     map[string]string
	attrs        map[string]map[string]string // queueURL → Policy 以外の属性
	raw          map[string]bool
	subs         map[string]string // subscriptionArn → queue name
	inbox        map[string][]map[string]any
	baseURL      string
}

func newAWSFake(t *testing.T) *awsFake {
	t.Helper()

	f := &awsFake{t: t, handlers: map[string]fakeHandler{}}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)

	return f
}

func (f *awsFake) on(op string, h fakeHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.handlers[op] = h
}

func (f *awsFake) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.calls...)
}

func (f *awsFake) serve(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		f.t.Errorf("fake: body の読み取りに失敗: %v", err)
	}

	req := parseFakeRequest(r, body)

	f.mu.Lock()
	h := f.handlers[req.op]
	f.calls = append(f.calls, req.op)
	f.mu.Unlock()

	var resp fakeResponse
	switch {
	case h != nil:
		resp = h(req)
	case req.form != nil:
		f.t.Errorf("fake: 未登録の操作 %q", req.op)
		resp = xmlErr("InvalidAction", "no fake handler")
	default:
		f.t.Errorf("fake: 未登録の操作 %q", req.op)
		resp = jsonErr("InvalidAction", "no fake handler")
	}

	w.Header().Set("Content-Type", resp.contentType)
	w.WriteHeader(resp.status)
	_, _ = w.Write([]byte(resp.body))
}

func parseFakeRequest(r *http.Request, body []byte) fakeRequest {
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		req := fakeRequest{op: target[strings.LastIndex(target, ".")+1:], json: map[string]any{}}
		_ = json.Unmarshal(body, &req.json)

		return req
	}

	form, _ := url.ParseQuery(string(body))

	return fakeRequest{op: form.Get("Action"), form: form}
}

// clients は、fake を向いた 3 クライアントを組み立てます。
func (f *awsFake) clients(t *testing.T) clients {
	t.Helper()

	c, err := newClients(t.Context(), options{
		dynamoDBEndpoint: f.server.URL,
		goAWSEndpoint:    f.server.URL,
		region:           defaultRegion,
	})
	require.NoError(t, err)

	return c
}

func jsonOK(v any) fakeResponse {
	b, err := json.Marshal(v)
	if err != nil {
		return fakeResponse{status: http.StatusInternalServerError, contentType: "text/plain", body: err.Error()}
	}

	return fakeResponse{status: http.StatusOK, contentType: "application/x-amz-json-1.0", body: string(b)}
}

func jsonErr(code, msg string) fakeResponse {
	return fakeResponse{
		status:      http.StatusBadRequest,
		contentType: "application/x-amz-json-1.0",
		body:        `{"__type":"` + code + `","message":"` + msg + `"}`,
	}
}

func xmlOK(op, inner string) fakeResponse {
	return fakeResponse{
		status:      http.StatusOK,
		contentType: "text/xml",
		body: `<` + op + `Response xmlns="http://sns.amazonaws.com/doc/2010-03-31/"><` + op + `Result>` + inner +
			`</` + op + `Result><ResponseMetadata><RequestId>req-1</RequestId></ResponseMetadata></` + op + `Response>`,
	}
}

func xmlErr(code, msg string) fakeResponse {
	return fakeResponse{
		status:      http.StatusBadRequest,
		contentType: "text/xml",
		body: `<ErrorResponse xmlns="http://sns.amazonaws.com/doc/2010-03-31/"><Error><Type>Sender</Type><Code>` + code +
			`</Code><Message>` + msg + `</Message></Error><RequestId>req-1</RequestId></ErrorResponse>`,
	}
}

// jsonPath は、入れ子の map を key で辿ります。途中で map でなくなれば nil を返します。
func jsonPath(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}

		v = m[k]
	}

	return v
}

// ---- DynamoDB ----

func installDynamoDB(f *awsFake) *ddbState {
	st := &ddbState{keys: map[string]struct{}{}}

	f.on("CreateTable", func(fakeRequest) fakeResponse {
		return jsonOK(map[string]any{"TableDescription": map[string]any{"TableStatus": "ACTIVE"}})
	})
	f.on("DescribeTable", func(fakeRequest) fakeResponse {
		return jsonOK(map[string]any{"Table": map[string]any{"TableStatus": "ACTIVE"}})
	})
	f.on("PutItem", st.putItem)
	f.on("Query", fakeQuery)
	f.on("UpdateTimeToLive", func(fakeRequest) fakeResponse {
		return jsonOK(map[string]any{"TimeToLiveSpecification": map[string]any{"AttributeName": attrExpires, "Enabled": true}})
	})
	f.on("DescribeTimeToLive", ttlDescription(attrExpires, "ENABLED"))
	f.on("DeleteTable", func(fakeRequest) fakeResponse {
		return jsonOK(map[string]any{"TableDescription": map[string]any{"TableStatus": "DELETING"}})
	})

	return st
}

func ttlDescription(attr, status string) fakeHandler {
	return func(fakeRequest) fakeResponse {
		return jsonOK(map[string]any{"TimeToLiveDescription": map[string]any{"AttributeName": attr, "TimeToLiveStatus": status}})
	}
}

func (st *ddbState) putItem(req fakeRequest) fakeResponse {
	seq, _ := jsonPath(req.json, "Item", attrSequence, "N").(string)

	st.mu.Lock()
	defer st.mu.Unlock()

	if _, dup := st.keys[seq]; dup {
		return jsonErr(fakeErrCondition, "The conditional request failed")
	}

	st.keys[seq] = struct{}{}

	return jsonOK(map[string]any{})
}

func ddbItems(seqs ...int) []map[string]any {
	items := make([]map[string]any, 0, len(seqs))
	for _, s := range seqs {
		items = append(items, map[string]any{
			attrStream:   map[string]any{"S": streamID},
			attrSequence: map[string]any{"N": strconv.Itoa(s)},
		})
	}

	return items
}

func ddbPage(seqs []int, last int) fakeResponse {
	out := map[string]any{"Items": ddbItems(seqs...)}
	if last > 0 {
		out["LastEvaluatedKey"] = ddbItems(last)[0]
	}

	return jsonOK(out)
}

// fakeQuery は、検査が投げる 4 種の Query に seed 1..5 を前提に答えます。
func fakeQuery(req fakeRequest) fakeResponse {
	kce, _ := req.json["KeyConditionExpression"].(string)
	forward := true
	if v, ok := req.json["ScanIndexForward"].(bool); ok {
		forward = v
	}

	limit, _ := req.json["Limit"].(float64)
	start, _ := jsonPath(req.json, "ExclusiveStartKey", attrSequence, "N").(string)

	switch {
	case strings.Contains(kce, "#seq"):
		return ddbPage([]int{3, 4, 5}, 0)
	case !forward:
		return ddbPage([]int{5}, 0)
	case limit == pageLimit && start == "":
		return ddbPage([]int{1, 2}, 2)
	case limit == pageLimit && start == "2":
		return ddbPage([]int{3, 4}, 4)
	case limit == pageLimit:
		return ddbPage([]int{5}, 0)
	default:
		return ddbPage([]int{1, 2, 3, 4, 5}, 0)
	}
}

// ---- GoAWS（SNS / SQS）----

func installGoAWS(f *awsFake) *goawsState {
	st := &goawsState{
		policies: map[string]string{},
		attrs:    map[string]map[string]string{},
		raw:      map[string]bool{},
		subs:     map[string]string{},
		inbox:    map[string][]map[string]any{},
		baseURL:  f.server.URL,
	}

	f.on("ListQueues", func(fakeRequest) fakeResponse { return jsonOK(map[string]any{"QueueUrls": []string{}}) })
	f.on("CreateQueue", st.createQueue)
	f.on("GetQueueAttributes", st.getQueueAttributes)
	f.on("SetQueueAttributes", st.setQueueAttributes)
	f.on("ReceiveMessage", st.receiveMessage)
	f.on("DeleteMessage", func(fakeRequest) fakeResponse { return jsonOK(map[string]any{}) })
	f.on("DeleteQueue", func(fakeRequest) fakeResponse { return jsonOK(map[string]any{}) })

	f.on("CreateTopic", func(req fakeRequest) fakeResponse {
		return xmlOK("CreateTopic", "<TopicArn>"+fakeTopic+req.form.Get("Name")+"</TopicArn>")
	})
	f.on("Subscribe", st.subscribe)
	f.on("SetSubscriptionAttributes", st.setSubscriptionAttributes)
	f.on("GetSubscriptionAttributes", st.getSubscriptionAttributes)
	f.on("ListSubscriptionsByTopic", st.listSubscriptionsByTopic)
	f.on("Publish", st.publish)
	f.on("Unsubscribe", func(fakeRequest) fakeResponse { return xmlOK("Unsubscribe", "") })
	f.on("DeleteTopic", func(fakeRequest) fakeResponse { return xmlOK("DeleteTopic", "") })

	return st
}

func (st *goawsState) queueURL(name string) string {
	return st.baseURL + "/" + fakeAccount + "/" + name
}

func queueName(urlOrArn string) string {
	if i := strings.LastIndexAny(urlOrArn, "/:"); i >= 0 {
		return urlOrArn[i+1:]
	}

	return urlOrArn
}

func (st *goawsState) createQueue(req fakeRequest) fakeResponse {
	name, _ := req.json["QueueName"].(string)
	if policy, ok := jsonPath(req.json, "Attributes", "Policy").(string); ok {
		if st.rejectPolicy {
			return jsonErr(fakeErrPolicy, "An invalid or out-of-range value was supplied for the input parameter.")
		}

		st.mu.Lock()
		st.policies[st.queueURL(name)] = policy
		st.mu.Unlock()
	}

	return jsonOK(map[string]any{"QueueUrl": st.queueURL(name)})
}

func (st *goawsState) getQueueAttributes(req fakeRequest) fakeResponse {
	queueURL, _ := req.json["QueueUrl"].(string)
	names, _ := req.json["AttributeNames"].([]any)
	attrs := map[string]string{}

	st.mu.Lock()
	defer st.mu.Unlock()

	for _, n := range names {
		name, _ := n.(string)
		switch name {
		case "QueueArn":
			attrs["QueueArn"] = fakeQueue + queueName(queueURL)
		case "Policy":
			if p, ok := st.policies[queueURL]; ok {
				attrs["Policy"] = p
			}
		default:
			if v, ok := st.attrs[queueURL][name]; ok {
				attrs[name] = v
			}
		}
	}

	return jsonOK(map[string]any{"Attributes": attrs})
}

// setQueueAttributes は、Policy を policies に、それ以外を attrs に保存します。実物の GoAWS と同じく
// rejectPolicy のときは Policy を拒否します。
func (st *goawsState) setQueueAttributes(req fakeRequest) fakeResponse {
	queueURL, _ := req.json["QueueUrl"].(string)
	given, _ := req.json["Attributes"].(map[string]any)

	st.mu.Lock()
	defer st.mu.Unlock()

	for name, v := range given {
		value, _ := v.(string)
		if name == "Policy" {
			if st.rejectPolicy {
				return jsonErr(fakeErrPolicy, "An invalid or out-of-range value was supplied for the input parameter.")
			}

			st.policies[queueURL] = value

			continue
		}

		if st.attrs[queueURL] == nil {
			st.attrs[queueURL] = map[string]string{}
		}

		st.attrs[queueURL][name] = value
	}

	return jsonOK(map[string]any{})
}

// listSubscriptionsByTopic は、購読中の queue を member として返します（endpoint は queue ARN）。
func (st *goawsState) listSubscriptionsByTopic(req fakeRequest) fakeResponse {
	topicArn := req.form.Get("TopicArn")

	st.mu.Lock()
	defer st.mu.Unlock()

	var members strings.Builder
	for subArn, name := range st.subs {
		members.WriteString("<member><SubscriptionArn>" + subArn + "</SubscriptionArn><Protocol>sqs</Protocol><Endpoint>" +
			fakeQueue + name + "</Endpoint><TopicArn>" + topicArn + "</TopicArn></member>")
	}

	return xmlOK("ListSubscriptionsByTopic", "<Subscriptions>"+members.String()+"</Subscriptions>")
}

func (st *goawsState) receiveMessage(req fakeRequest) fakeResponse {
	queueURL, _ := req.json["QueueUrl"].(string)

	st.mu.Lock()
	defer st.mu.Unlock()

	msgs := st.inbox[queueURL]
	st.inbox[queueURL] = nil

	return jsonOK(map[string]any{"Messages": msgs})
}

func (st *goawsState) subscribe(req fakeRequest) fakeResponse {
	name := queueName(req.form.Get("Endpoint"))
	subArn := req.form.Get("TopicArn") + ":sub-" + name

	st.mu.Lock()
	st.subs[subArn] = name
	st.mu.Unlock()

	return xmlOK("Subscribe", "<SubscriptionArn>"+subArn+"</SubscriptionArn>")
}

func (st *goawsState) setSubscriptionAttributes(req fakeRequest) fakeResponse {
	if req.form.Get("AttributeName") == attrRawMessageDelivery {
		st.mu.Lock()
		st.raw[req.form.Get("SubscriptionArn")] = req.form.Get("AttributeValue") == "true"
		st.mu.Unlock()
	}

	return xmlOK("SetSubscriptionAttributes", "")
}

func (st *goawsState) getSubscriptionAttributes(req fakeRequest) fakeResponse {
	st.mu.Lock()
	raw := st.raw[req.form.Get("SubscriptionArn")]
	st.mu.Unlock()

	return xmlOK("GetSubscriptionAttributes",
		"<Attributes><entry><key>"+attrRawMessageDelivery+"</key><value>"+strconv.FormatBool(raw)+"</value></entry></Attributes>")
}

// publish は、購読中の queue へ配送します。RawMessageDelivery が無効な subscription には SNS envelope で届けます。
func (st *goawsState) publish(req fakeRequest) fakeResponse {
	message := req.form.Get("Message")
	attrs := map[string]any{}
	for i := 1; ; i++ {
		prefix := "MessageAttributes.entry." + strconv.Itoa(i)
		name := req.form.Get(prefix + ".Name")
		if name == "" {
			break
		}

		attrs[name] = map[string]any{
			"DataType":    req.form.Get(prefix + ".Value.DataType"),
			"StringValue": req.form.Get(prefix + ".Value.StringValue"),
		}
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	for subArn, name := range st.subs {
		body := message
		if !st.raw[subArn] {
			body = `{"Type":"Notification","Message":` + strconv.Quote(message) + `}`
		}

		st.inbox[st.queueURL(name)] = append(st.inbox[st.queueURL(name)], map[string]any{
			"MessageId":         "m-" + name,
			"ReceiptHandle":     "r-" + name,
			"Body":              body,
			"MessageAttributes": attrs,
		})
	}

	return xmlOK("Publish", "<MessageId>m-1</MessageId>")
}
