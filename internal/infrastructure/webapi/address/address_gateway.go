// Package address は、郵便番号住所補完 gateway の外部サービス実装を提供します。
package address

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/address"
	"go-boilerplate/pkg/xerrors"
)

// downstream は、profile / breaker / metrics / budget の論理依存名です。
const downstream httpclient.Downstream = "address"

// zipcloudSuccessStatus は、zipcloud レスポンス body の status が示す正常値です。
// zipcloud は HTTP は 200 を返しつつ body の status で結果を表すため、body 側で判定します。
const zipcloudSuccessStatus = 200

// maxCandidates は、外部応答から取り込む住所候補の上限件数です。認証不要エンドポイントが
// 外部応答を信頼し切って無制限にループ処理しないための防御的上限（1 郵便番号あたり実際は数件）。
const maxCandidates = 50

// Endpoint は、外部郵便番号 lookup サービスのベース URL です（DI で注入）。
type Endpoint string

// gateway は、boundary.Gateway の外部サービス実装です。
type gateway struct {
	endpoint Endpoint
	client   httpclient.Client
	tracer   observability.LayerTracer
}

// zipcloudResponse は、外部 API（zipcloud）の JSON レスポンスの形を表します。
type zipcloudResponse struct {
	Status  int              `json:"status"`
	Message *string          `json:"message"`
	Results []zipcloudResult `json:"results"`
}

// zipcloudResult は、zipcloud の住所候補 1 件の形を表します。
// address1 が都道府県フル表記（例 "東京都"）、address2 が市区町村、address3 が町域です。
type zipcloudResult struct {
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	Address3 string `json:"address3"`
}

// NewDownstreamProfile は、外部郵便番号 lookup サービス向けの resilient プロファイルを返します。
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

// New は、郵便番号住所補完 gateway の外部サービス実装を生成します。
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

// Lookup は、外部 API から郵便番号に対応する住所候補を取得し、境界 DTO へ変換して返します。
// 外部レスポンスの型・エラーは本メソッド内で完結し、外側へは boundary.Candidate と apperror のみ露出します。
func (g *gateway) Lookup(ctx context.Context, postalCode string) ([]*boundary.Candidate, error) {
	ctx, endSpan := g.tracer.Start(ctx)
	defer endSpan()

	reqURL := fmt.Sprintf("%s/api/search?zipcode=%s", g.endpoint, url.QueryEscape(postalCode))

	resp, err := g.client.Do(ctx, httpclient.NewRequest(httpclient.MethodGet(), downstream, reqURL))
	if err != nil {
		return nil, err // substrate が apperror sentinel へ写像済み
	}

	var body zipcloudResponse
	if uerr := json.Unmarshal(resp.Body, &body); uerr != nil {
		return nil, xerrors.Wrap(apperror.ErrUnavailable, "invalid address lookup response: "+uerr.Error())
	}
	if body.Status != zipcloudSuccessStatus {
		msg := ""
		if body.Message != nil {
			msg = *body.Message
		}
		return nil, xerrors.Wrap(apperror.ErrUnavailable,
			fmt.Sprintf("address lookup returned status %d: %s", body.Status, msg))
	}

	// results が null（該当なし）でも障害ではないため、空スライス + nil error で返す（degrade 対象外）。
	results := body.Results
	if len(results) > maxCandidates {
		results = results[:maxCandidates]
	}
	candidates := make([]*boundary.Candidate, 0, len(results))
	for _, r := range results {
		candidates = append(candidates, &boundary.Candidate{
			PrefectureName: r.Address1,
			City:           r.Address2,
			Town:           r.Address3,
		})
	}
	return candidates, nil
}
