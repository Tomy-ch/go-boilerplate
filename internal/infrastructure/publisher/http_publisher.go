// Package publisher は、ドメインイベントの publish 境界（boundary publisher.Publisher）の
// HTTP 実装を提供します。relay engine が claim した outbox メッセージを受信エンドポイントへ POST します。
package publisher

import (
	"context"

	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/publisher"
)

// downstream は、profile / breaker / metrics / budget の論理依存名です。
const downstream httpclient.Downstream = "outbox"

// Endpoint は、メッセージの送信先エンドポイント URL です（構築時に config から固定注入）。
type Endpoint string

// httpPublisher は、boundary.Publisher の HTTP 実装です。
type httpPublisher struct {
	endpoint Endpoint
	client   httpclient.Client
	tracer   observability.LayerTracer
}

// NewDownstreamProfile は、outbox publish 向けの resilient プロファイルを返します。
// relay の poll ループ自体が at-least-once の retry 本体であるため、transport 層 retry は
// 二重になる。これを避けるため MaxAttempts=1 で transport retry を無効化する（D10）。
// traceparent は emit 時に capture した値を headers で明示伝搬するため、ここでの自動 inject は抑止する。
func NewDownstreamProfile() httpclient.DownstreamProfile {
	p := httpclient.DefaultProfile()
	p.MaxAttempts = 1
	p.PropagateTrace = false
	return httpclient.DownstreamProfile{Name: downstream, Profile: p}
}

// New は、publish 境界の HTTP 実装を生成します。
func New(
	endpoint Endpoint,
	client httpclient.Client,
	tf observability.TracerFactory,
) boundary.Publisher {
	return &httpPublisher{
		endpoint: endpoint,
		client:   client,
		tracer:   tf.Infra(),
	}
}

// Publish は、メッセージを受信エンドポイントへ POST します。
// message_id を Idempotency-Key として伝搬し、非冪等メソッドだが retry は明示的に無効化します
// （再送は relay の次 poll が担う）。非 2xx / transport 失敗は substrate が apperror へ写像して返します。
func (p *httpPublisher) Publish(ctx context.Context, m boundary.Message) error {
	ctx, endSpan := p.tracer.Start(ctx)
	defer endSpan()

	header := httpclient.Header{"Content-Type": {"application/json"}}
	for k, v := range m.Headers {
		header[k] = []string{v}
	}

	_, err := p.client.Do(ctx, &httpclient.Request{
		Downstream:     downstream,
		Method:         httpclient.MethodPost,
		URL:            string(p.endpoint),
		Header:         header,
		Body:           m.Payload,
		IdempotencyKey: m.MessageID.String(),
		AllowRetry:     false,
	})
	return err
}
