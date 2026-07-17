// Package exchangerate は、為替レート gateway の外部サービス実装を提供します。
package exchangerate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
	"go-boilerplate/pkg/xerrors"
)

// downstream は、profile / breaker / metrics / budget の論理依存名です。
const downstream httpclient.Downstream = "exchangerate"

// Endpoint は、外部為替レートサービスのベース URL です（DI で注入）。
type Endpoint string

// gateway は、boundary.Gateway の外部サービス実装です。
type gateway struct {
	endpoint Endpoint
	client   httpclient.Client
	tracer   observability.LayerTracer
}

// rateResponse は、外部 API の JSON レスポンスの形を表します。
type rateResponse struct {
	Rate float64 `json:"rate"`
}

// NewDownstreamProfile は、外部為替サービス向けの resilient プロファイルを返します。
// 外部サービスのため trace を伝搬せず、private/loopback 宛て接続を拒否します。
func NewDownstreamProfile() httpclient.DownstreamProfile {
	p := httpclient.DefaultProfile()
	p.PropagateTrace = false
	p.AllowPrivateNetwork = false
	return httpclient.DownstreamProfile{Name: downstream, Profile: p}
}

// RequiredDownstream は、本 gateway が使用する Downstream を返します。
// required_downstreams へ供給することで、対応 profile が未登録のまま起動した場合に
// silent な DefaultProfile fallback ではなく loud な起動失敗になることを保証します。
func RequiredDownstream() httpclient.Downstream {
	return downstream
}

// New は、為替レート gateway の外部サービス実装を生成します。
func New(
	endpoint Endpoint,
	client httpclient.Client,
	tf observability.TracerFactory,
) boundary.Gateway {
	return &gateway{
		endpoint: endpoint,
		client:   client,
		tracer:   tf.Infra(),
	}
}

// GetRate は、外部 API から為替レートを取得し、境界 DTO へ変換して返します。
func (g *gateway) GetRate(ctx context.Context, base, quote string) (*boundary.Rate, error) {
	ctx, endSpan := g.tracer.Start(ctx)
	defer endSpan()

	reqURL := fmt.Sprintf("%s/rates?base=%s&quote=%s",
		g.endpoint, url.QueryEscape(base), url.QueryEscape(quote))

	resp, err := g.client.Do(ctx, httpclient.NewRequest(httpclient.MethodGet(), downstream, reqURL))
	if err != nil {
		return nil, err // substrate が apperror sentinel へ写像済み
	}

	var body rateResponse
	if uerr := json.Unmarshal(resp.Body, &body); uerr != nil {
		return nil, xerrors.Wrap(apperror.ErrUnavailable, "invalid exchangerate response: "+uerr.Error())
	}
	if body.Rate <= 0 {
		return nil, xerrors.Wrap(apperror.ErrUnavailable, "exchangerate response has non-positive rate")
	}

	return &boundary.Rate{Base: base, Quote: quote, Value: body.Rate}, nil
}
