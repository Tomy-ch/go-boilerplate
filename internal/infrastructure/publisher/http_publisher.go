// Package publisher は、ドメインイベントの publish 境界（boundary publisher.Publisher）の
// HTTP 実装を提供します。relay engine が claim した outbox メッセージを受信エンドポイントへ POST します。
package publisher

import (
	"context"

	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/publisher"
	"go-boilerplate/pkg/httpheader"
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
// transport retry の無効化・trace 自動 inject の抑止・private/loopback 宛ての拒否は、いずれも
// README.md の Design Policy（D10）に理由があります。
func NewDownstreamProfile() httpclient.DownstreamProfile {
	p := httpclient.DefaultProfile()
	p.MaxAttempts = 1
	p.PropagateTrace = false
	p.AllowPrivateNetwork = false
	return httpclient.DownstreamProfile{Name: downstream, Profile: p}
}

// RequiredDownstream は、本 publisher が使用する Downstream を返します。
// required_downstreams へ供給することで、対応 profile が未登録のまま起動した場合に
// silent な DefaultProfile fallback ではなく loud な起動失敗になることを保証します。
func RequiredDownstream() httpclient.Downstream {
	return downstream
}

// NewHTTP は、publish 境界の HTTP 実装を生成します。
func NewHTTP(
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
// message_id を Idempotency-Key として伝搬します。非 2xx / transport 失敗は substrate が
// apperror へ写像して返します。
func (p *httpPublisher) Publish(ctx context.Context, m boundary.Message) error {
	ctx, endSpan := p.tracer.Start(ctx)
	defer endSpan()

	header := httpclient.Header{"Content-Type": {"application/json"}}
	for k, v := range m.Headers {
		// emit 側と同じ判定だが、emit を経由しない行への防御として必要。
		if httpheader.IsSensitive(k) {
			continue
		}
		header[k] = []string{v}
	}

	_, err := p.client.Do(ctx, httpclient.NewRequest(
		httpclient.MethodPost(), downstream, string(p.endpoint),
		httpclient.WithHeader(header),
		httpclient.WithBody(m.Payload),
		httpclient.WithIdempotencyKey(m.MessageID.String()),
	))
	return err
}
